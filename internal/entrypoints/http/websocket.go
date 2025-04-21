package http

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2/log"
	"github.com/gofiber/websocket/v2"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
)

func (r *Router) removeTrack(t *webrtc.TrackLocalStaticRTP, join_url string) {
	r.rooms[join_url].listLock.Lock()
	defer func() {
		r.rooms[join_url].listLock.Unlock()
		r.signalPeerConnections(join_url)
	}()

	delete(r.rooms[join_url].trackLocals, t.ID())
}
func (r *Router) addTrack(t *webrtc.TrackRemote, join_url string) *webrtc.TrackLocalStaticRTP {
	r.rooms[join_url].listLock.Lock()
	defer func() {
		r.rooms[join_url].listLock.Unlock()
		r.signalPeerConnections(join_url)
	}()

	// Create a new TrackLocal with the same codec as our incoming
	trackLocal, err := webrtc.NewTrackLocalStaticRTP(t.Codec().RTPCodecCapability, t.ID(), t.StreamID())
	if err != nil {
		panic(err)
	}

	r.rooms[join_url].trackLocals[t.ID()] = trackLocal
	return trackLocal
}
func (r *Router) dispatchKeyFrame(join_url string) {
	r.rooms[join_url].listLock.Lock()
	defer r.rooms[join_url].listLock.Unlock()

	for i := range r.rooms[join_url].peerConnections {
		for _, receiver := range r.rooms[join_url].peerConnections[i].peerConnection.GetReceivers() {
			if receiver.Track() == nil {
				continue
			}
			if receiver.Track() == nil {
				continue
			}
			_ = r.rooms[join_url].peerConnections[i].peerConnection.WriteRTCP([]rtcp.Packet{
				&rtcp.PictureLossIndication{
					MediaSSRC: uint32(receiver.Track().SSRC()),
				},
			})
		}
	}
}

func (r *Router) signalPeerConnections(join_url string) {
	r.rooms[join_url].listLock.Lock()
	defer func() {
		r.rooms[join_url].listLock.Unlock()
		r.dispatchKeyFrame(join_url)
	}()

	attemptSync := func() (tryAgain bool) {
		for i := range r.rooms[join_url].peerConnections {
			if r.rooms[join_url].peerConnections[i].peerConnection.ConnectionState() == webrtc.PeerConnectionStateClosed {
				r.rooms[join_url].peerConnections = append(r.rooms[join_url].peerConnections[:i], r.rooms[join_url].peerConnections[i+1:]...)
				return true // We modified the slice, start from the beginning
			}

			// map of sender we already are seanding, so we don't double send
			existingSenders := map[string]bool{}

			for _, sender := range r.rooms[join_url].peerConnections[i].peerConnection.GetSenders() {
				if sender.Track() == nil {
					continue
				}

				existingSenders[sender.Track().ID()] = true

				// If we have a RTPSender that doesn't map to a existing track remove and signal
				if _, ok := r.rooms[join_url].trackLocals[sender.Track().ID()]; !ok {
					if err := r.rooms[join_url].peerConnections[i].peerConnection.RemoveTrack(sender); err != nil {
						return true
					}
				}
			}

			// Don't receive videos we are sending, make sure we don't have loopback
			// for _, receiver := range peerConnections[i].peerConnection.GetReceivers() {
			// 	if receiver.Track() == nil {
			// 		continue
			// 	}

			// 	existingSenders[receiver.Track().ID()] = true
			// }

			// Add all track we aren't sending yet to the PeerConnection
			for trackID := range r.rooms[join_url].trackLocals {
				if _, ok := existingSenders[trackID]; !ok {
					if _, err := r.rooms[join_url].peerConnections[i].peerConnection.AddTrack(r.rooms[join_url].trackLocals[trackID]); err != nil {
						return true
					}
				}
			}

			offer, err := r.rooms[join_url].peerConnections[i].peerConnection.CreateOffer(nil)
			if err != nil {
				return true
			}

			if err = r.rooms[join_url].peerConnections[i].peerConnection.SetLocalDescription(offer); err != nil {
				return true
			}

			offerString, err := json.Marshal(offer)
			if err != nil {
				log.Errorf("Failed to marshal offer to json: %v", err)
				return true
			}

			log.Infof("Send offer to client: %v", offer)

			if err = r.rooms[join_url].peerConnections[i].websocket.WriteJSON(&websocketMessage{
				Event: "offer",
				Data:  string(offerString),
			}); err != nil {
				return true
			}
		}

		return
	}

	for syncAttempt := 0; ; syncAttempt++ {
		if syncAttempt == 25 {
			// Release the lock and attempt a sync in 3 seconds. We might be blocking a RemoveTrack or AddTrack
			go func() {
				time.Sleep(time.Second * 3)
				r.signalPeerConnections(join_url)
			}()
			return
		}

		if !attemptSync() {
			break
		}
	}
}

func (r *Router) WebSocketHandler(ws *websocket.Conn) {
	join_url := ws.Query("join_url", "error")
	if join_url == "error" {
		return
	}
	if _, ok := r.rooms[join_url]; !ok {
		r.rooms[join_url] = &Room{
			trackLocals:     make(map[string]*webrtc.TrackLocalStaticRTP),
			peerConnections: make([]peerConnectionState, 0),
		}
	}
	c := &threadSafeWriter{Conn: ws, Mutex: sync.Mutex{}}
	defer c.Close()
	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		log.Errorf("Failed to creates a PeerConnection: %v", err)
		return
	}
	defer peerConnection.Close() //nolint
	for _, typ := range []webrtc.RTPCodecType{webrtc.RTPCodecTypeVideo, webrtc.RTPCodecTypeAudio} {
		if _, err := peerConnection.AddTransceiverFromKind(typ, webrtc.RTPTransceiverInit{
			Direction: webrtc.RTPTransceiverDirectionRecvonly,
		}); err != nil {
			log.Errorf("Failed to add transceiver: %v", err)
			return
		}
	}
	r.rooms[join_url].listLock.Lock()
	r.rooms[join_url].peerConnections = append(r.rooms[join_url].peerConnections, peerConnectionState{peerConnection: peerConnection, websocket: c})
	r.rooms[join_url].listLock.Unlock()
	peerConnection.OnICECandidate(func(i *webrtc.ICECandidate) {
		if i == nil {
			return
		}
		candidateString, err := json.Marshal(i.ToJSON())
		if err != nil {
			log.Errorf("Failed to marshal candidate to json: %v", err)
			return
		}
		if err = c.WriteJSON(&websocketMessage{
			Event: "candidate",
			Data:  string(candidateString),
		}); err != nil {
			log.Errorf("Failed to send candidate to peer: %v", err)
		}
	})
	peerConnection.OnConnectionStateChange(func(p webrtc.PeerConnectionState) {
		log.Infof("Connection state change: %s", p)

		switch p {
		case webrtc.PeerConnectionStateFailed:
			if err := peerConnection.Close(); err != nil {
				log.Errorf("Failed to close PeerConnection: %v", err)
			}
		case webrtc.PeerConnectionStateClosed:
			r.signalPeerConnections(join_url)
		default:
		}
	})
	peerConnection.OnTrack(func(t *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		log.Infof("Got remote track: Kind=%s, ID=%s, PayloadType=%d", t.Kind(), t.ID(), t.PayloadType())

		// Create a track to fan out our incoming video to all peers
		trackLocal := r.addTrack(t, join_url)
		defer r.removeTrack(trackLocal, join_url)

		buf := make([]byte, 1500)
		rtpPkt := &rtp.Packet{}

		for {
			i, _, err := t.Read(buf)
			if err != nil {
				return
			}

			if err = rtpPkt.Unmarshal(buf[:i]); err != nil {
				log.Errorf("Failed to unmarshal incoming RTP packet: %v", err)
				return
			}

			rtpPkt.Extension = false
			rtpPkt.Extensions = nil

			if err = trackLocal.WriteRTP(rtpPkt); err != nil {
				return
			}
		}
	})
	peerConnection.OnICEConnectionStateChange(func(is webrtc.ICEConnectionState) {
		log.Infof("ICE connection state changed: %s", is)
	})
	r.signalPeerConnections(join_url)
	message := &websocketMessage{}
	for {
		_, raw, err := c.ReadMessage()
		if err != nil {
			//log.Errorf("Failed to read message: %v", err)
			return
		}

		log.Infof("Got message: %s", raw)

		if err := json.Unmarshal(raw, &message); err != nil {
			log.Errorf("Failed to unmarshal json to message: %v", err)
			return
		}

		switch message.Event {
		case "candidate":
			candidate := webrtc.ICECandidateInit{}
			if err := json.Unmarshal([]byte(message.Data), &candidate); err != nil {
				log.Errorf("Failed to unmarshal json to candidate: %v", err)
				return
			}

			log.Infof("Got candidate: %v", candidate)

			if err := peerConnection.AddICECandidate(candidate); err != nil {
				log.Errorf("Failed to add ICE candidate: %v", err)
				return
			}
		case "answer":
			answer := webrtc.SessionDescription{}
			if err := json.Unmarshal([]byte(message.Data), &answer); err != nil {
				log.Errorf("Failed to unmarshal json to answer: %v", err)
				return
			}

			log.Infof("Got answer: %v", answer)

			if err := peerConnection.SetRemoteDescription(answer); err != nil {
				log.Errorf("Failed to set remote description: %v", err)
				return
			}
		default:
			log.Errorf("unknown message: %+v", message)
		}
	}
}

//func NewWebSocket(log logging.LeveledLogger) *WebSocket {
//	ws := &WebSocket{}
//
//	go ws.dispatchKeyFrames()
//	return ws
//}

//func (ws *WebSocket) HandleWebsocket(c *websocket.Conn) {
//	defer func() {
//		ws.mu.Lock()
//		defer ws.mu.Unlock()
//		for i := range ws.peers {
//			if ws.peers[i].wsConn == c {
//				ws.peers = append(ws.peers[:i], ws.peers[i+1:]...)
//				break
//			}
//		}
//		c.Close()
//	}()
//
//	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{})
//	if err != nil {
//		ws.log.Errorf("Failed to create peer connection: %v", err)
//		return
//	}
//	defer peerConnection.Close()
//
//	ws.mu.Lock()
//	ws.peers = append(ws.peers, peerConnectionState{
//		peerConnection: peerConnection,
//		wsConn:         c,
//	})
//	ws.mu.Unlock()
//
//	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
//		if candidate == nil {
//			return
//		}
//
//		candidateJSON, err := json.Marshal(candidate.ToJSON())
//		if err != nil {
//			ws.log.Errorf("Failed to marshal candidate: %v", err)
//			return
//		}
//
//		msg := websocketMessage{
//			Event: "candidate",
//			Data:  candidateJSON,
//		}
//
//		if err := c.WriteJSON(msg); err != nil {
//			ws.log.Errorf("Failed to send candidate: %v", err)
//		}
//	})
//
//	peerConnection.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
//		trackLocal := ws.addTrack(track)
//		defer ws.removeTrack(trackLocal)
//
//		buf := make([]byte, 1500)
//		rtpPacket := &rtp.Packet{}
//		for {
//			n, _, err := track.Read(buf)
//			if err != nil {
//				return
//			}
//
//			if err := rtpPacket.Unmarshal(buf[:n]); err != nil {
//				ws.log.Errorf("Failed to unmarshal RTP packet: %v", err)
//				continue
//			}
//
//			b, err := rtpPacket.Marshal()
//			if err != nil {
//				panic(err)
//			}
//			if _, err := trackLocal.Write(b); err != nil {
//				ws.log.Errorf("Failed to write RTP packet: %v", err)
//			}
//		}
//	})
//
//	peerConnection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
//		ws.log.Infof("Connection state changed: %s", state.String())
//	})
//
//	for {
//		var msg websocketMessage
//		if err := c.ReadJSON(&msg); err != nil {
//			ws.log.Errorf("Websocket read error: %v", err)
//			break
//		}
//
//		switch msg.Event {
//		case "offer":
//			var offer webrtc.SessionDescription
//			if err := json.Unmarshal(msg.Data, &offer); err != nil {
//				ws.log.Errorf("Failed to unmarshal offer: %v", err)
//				continue
//			}
//
//			if err := peerConnection.SetRemoteDescription(offer); err != nil {
//				ws.log.Errorf("Failed to set remote description: %v", err)
//				continue
//			}
//
//			answer, err := peerConnection.CreateAnswer(nil)
//			if err != nil {
//				ws.log.Errorf("Failed to create answer: %v", err)
//				continue
//			}
//
//			if err := peerConnection.SetLocalDescription(answer); err != nil {
//				ws.log.Errorf("Failed to set local description: %v", err)
//				continue
//			}
//
//			answerJSON, err := json.Marshal(answer)
//			if err != nil {
//				ws.log.Errorf("Failed to marshal answer: %v", err)
//				continue
//			}
//
//			resp := websocketMessage{
//				Event: "answer",
//				Data:  answerJSON,
//			}
//
//			if err := c.WriteJSON(resp); err != nil {
//				ws.log.Errorf("Failed to send answer: %v", err)
//			}
//
////		case "candidate":
////			var candidate webrtc.ICECandidateInit
////			if err := json.Unmarshal(msg.Data, &candidate); err != nil {
////				ws.log.Errorf("Failed to unmarshal candidate: %v", err)
////				continue
////			}
////
////			if err := peerConnection.AddICECandidate(candidate); err != nil {
////				ws.log.Errorf("Failed to add ICE candidate: %v", err)
////			}
////		}
////	}
////}
//
//func (ws *WebSocket) removeTrack(trackLocal *webrtc.TrackLocalStaticRTP) {
//	ws.mu.Lock()
//	defer ws.mu.Unlock()
//
//	delete(ws.trackLocals, trackLocal.ID())
//}
//
//func (ws *WebSocket) dispatchKeyFrames() {
//	for range ws.keyframeTicker.C {
//		ws.mu.RLock()
//		defer ws.mu.RUnlock()
//
//		for _, peer := range ws.peers {
//			for _, receiver := range peer.peerConnection.GetReceivers() {
//				if receiver.Track() == nil {
//					continue
//				}
//
//				_ = peer.peerConnection.WriteRTCP([]rtcp.Packet{
//					&rtcp.PictureLossIndication{
//						MediaSSRC: uint32(receiver.Track().SSRC()),
//					},
//				})
//			}
//		}
//	}
//}
//
//func (r *Router) WebsocketMiddleware(c *fiber.Ctx) error {
//	if websocket.IsWebSocketUpgrade(c) {
//		c.Locals("allowed", true)
//		return c.Next()
//	}
//	return fiber.ErrUpgradeRequired
//}
//
//func (ws *WebSocket) addTrack(track *webrtc.TrackRemote) *webrtc.TrackLocalStaticRTP {
//	ws.mu.Lock()
//	defer ws.mu.Unlock()
//
//	trackLocal, err := webrtc.NewTrackLocalStaticRTP(
//		track.Codec().RTPCodecCapability,
//		track.ID(),
//		track.StreamID(),
//	)
//	if err != nil {
//		ws.log.Errorf("Failed to create track local: %v", err)
//		return nil
//	}
//
//	ws.trackLocals[track.ID()] = trackLocal
//	return trackLocal
//}
