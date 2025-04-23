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
	if existingTrack, ok := r.rooms[join_url].trackLocals[t.ID()]; ok {
		return existingTrack
	}

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
		// Откладываем key frame запросы
		time.AfterFunc(100*time.Millisecond, func() {
			r.dispatchKeyFrame(join_url)
		})
	}()

	// Упрощенная и более эффективная логика синхронизации
	var wg sync.WaitGroup
	for i := 0; i < len(r.rooms[join_url].peerConnections); i++ {
		if r.rooms[join_url].peerConnections[i].peerConnection.ConnectionState() == webrtc.PeerConnectionStateClosed {
			r.rooms[join_url].peerConnections = append(r.rooms[join_url].peerConnections[:i], r.rooms[join_url].peerConnections[i+1:]...)
			i--
			continue
		}

		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			r.signalPeer(join_url, idx)
		}(i)
	}
	wg.Wait()
}

func (r *Router) signalPeer(join_url string, idx int) {
	pcState := r.rooms[join_url].peerConnections[idx]

	// Оптимизация: собираем все изменения перед созданием offer
	existingSenders := make(map[string]bool)
	for _, sender := range pcState.peerConnection.GetSenders() {
		if sender.Track() != nil {
			existingSenders[sender.Track().ID()] = true
			if _, ok := r.rooms[join_url].trackLocals[sender.Track().ID()]; !ok {
				if err := pcState.peerConnection.RemoveTrack(sender); err != nil {
					return
				}
			}
		}
	}

	for trackID := range r.rooms[join_url].trackLocals {
		if !existingSenders[trackID] {
			if _, err := pcState.peerConnection.AddTrack(r.rooms[join_url].trackLocals[trackID]); err != nil {
				return
			}
		}
	}

	offer, err := pcState.peerConnection.CreateOffer(nil)
	if err != nil {
		return
	}

	if err = pcState.peerConnection.SetLocalDescription(offer); err != nil {
		return
	}

	offerString, err := json.Marshal(offer)
	if err != nil {
		log.Errorf("Failed to marshal offer: %v", err)
		return
	}

	if err = pcState.websocket.WriteJSON(&websocketMessage{
		Event: "offer",
		Data:  string(offerString),
	}); err != nil {
		log.Errorf("Failed to send offer: %v", err)
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
		r.rooms[join_url].listLock.Lock()
		defer r.rooms[join_url].listLock.Unlock()
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
		log.Infof("Got remote track: Kind=%s, ID=%s", t.Kind(), t.ID())

		// Для аудио треков можно использовать более легковесную обработку
		//if t.Kind() == webrtc.RTPCodecTypeAudio {
		//	// Пропускаем аудио треки или обрабатываем их более эффективно
		//	return
		//}

		trackLocal := r.addTrack(t, join_url)
		if trackLocal == nil {
			return
		}
		defer r.removeTrack(trackLocal, join_url)

		// Используем буфер оптимального размера
		buf := make([]byte, 1450) // MTU typical size
		rtpPkt := &rtp.Packet{}

		for {
			i, _, err := t.Read(buf)
			if err != nil {
				log.Infof("Track reading stopped: %v", err)
				return
			}

			if err = rtpPkt.Unmarshal(buf[:i]); err != nil {
				log.Errorf("Failed to unmarshal RTP: %v", err)
				continue
			}

			// Минимизируем обработку пакета
			if err = trackLocal.WriteRTP(rtpPkt); err != nil {
				log.Infof("Track writing stopped: %v", err)
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

// Добавляем очистку неиспользуемых комнат
func (r *Router) cleanupRooms() {
	for {
		time.Sleep(5 * time.Minute)
		r.roomslock.Lock()
		for url, room := range r.rooms {
			if len(room.peerConnections) == 0 && len(room.trackLocals) == 0 {
				delete(r.rooms, url)
				log.Infof("Cleaned up unused room: %s", url)
			}
		}
		r.roomslock.Unlock()
	}
}

// Инициализируем в конструкторе Router
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
