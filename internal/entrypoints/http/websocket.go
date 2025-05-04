package http

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"GoSpeak/internal/model"

	"github.com/gofiber/fiber/v2/log"
	"github.com/gofiber/websocket/v2"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
)

var Config = webrtc.Configuration{
	ICEServers: []webrtc.ICEServer{
		webrtc.ICEServer{
			URLs: []string{"stun:stun.l.google.com:19302"},
		},
	},
}
var rtpBufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 1450)
	},
}

func (r *Router) removeTrack(t *webrtc.TrackLocalStaticRTP, joinUrl string) {
	r.rooms[joinUrl].listLock.Lock()
	defer func() {
		r.rooms[joinUrl].listLock.Unlock()
		r.signalPeerConnections(joinUrl)
	}()

	delete(r.rooms[joinUrl].trackLocals, t.ID())
}
func (r *Router) addTrack(t *webrtc.TrackRemote, joinUrl string) *webrtc.TrackLocalStaticRTP {
	r.rooms[joinUrl].listLock.Lock()
	defer func() {
		r.rooms[joinUrl].listLock.Unlock()
		r.signalPeerConnections(joinUrl)
	}()
	if existingTrack, ok := r.rooms[joinUrl].trackLocals[t.ID()]; ok {
		return existingTrack
	}

	trackLocal, err := webrtc.NewTrackLocalStaticRTP(t.Codec().RTPCodecCapability, t.ID(), t.StreamID())
	if err != nil {
		panic(err)
	}

	r.rooms[joinUrl].trackLocals[t.ID()] = trackLocal
	slog.Info(fmt.Sprintf("найдено %d подключений", len(r.rooms[joinUrl].trackLocals)))
	return trackLocal
}

func (r *Router) dispatchKeyFrame(joinUrl string) {
	r.rooms[joinUrl].listLock.Lock()
	defer r.rooms[joinUrl].listLock.Unlock()

	for i := range r.rooms[joinUrl].peerConnections {
		for _, receiver := range r.rooms[joinUrl].peerConnections[i].peerConnection.GetReceivers() {
			if receiver.Track() == nil {
				continue
			}

			_ = r.rooms[joinUrl].peerConnections[i].peerConnection.WriteRTCP([]rtcp.Packet{
				&rtcp.PictureLossIndication{
					MediaSSRC: uint32(receiver.Track().SSRC()),
				},
			})
		}
	}
}

func (r *Router) signalPeerConnections(joinUrl string) {
	r.rooms[joinUrl].listLock.Lock()
	defer func() {
		r.rooms[joinUrl].pendingSignal = false

		r.rooms[joinUrl].listLock.Unlock()

	}()
	if r.rooms[joinUrl].pendingSignal {
		return
	}
	r.rooms[joinUrl].pendingSignal = true
	time.AfterFunc(100*time.Millisecond, func() {
		var wg sync.WaitGroup
		r.roomslock.Lock()
		for i := 0; i < len(r.rooms[joinUrl].peerConnections); i++ {
			if r.rooms[joinUrl].peerConnections[i].peerConnection.ConnectionState() == webrtc.PeerConnectionStateClosed {
				r.rooms[joinUrl].peerConnections = append(r.rooms[joinUrl].peerConnections[:i], r.rooms[joinUrl].peerConnections[i+1:]...)
				i--
				continue
			}

			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				r.signalPeer(joinUrl, idx)
			}(i)
		}
		r.roomslock.Unlock()

		time.AfterFunc(10*time.Millisecond, func() {
			r.dispatchKeyFrame(joinUrl)
		})
		wg.Wait()
	})
}

func (r *Router) signalPeer(joinUrl string, idx int) {
	r.roomslock.Lock()
	r.rooms[joinUrl].listLock.Lock()
	pcState := r.rooms[joinUrl].peerConnections[idx]
	r.roomslock.Unlock()
	existingSenders := make(map[string]bool)
	for _, sender := range pcState.peerConnection.GetSenders() {
		if sender.Track() != nil {
			existingSenders[sender.Track().ID()] = true
			if _, ok := r.rooms[joinUrl].trackLocals[sender.Track().ID()]; !ok {
				if err := pcState.peerConnection.RemoveTrack(sender); err != nil {
					log.Errorf("Failed to remove track: %v", err)
				}
			}
		}
	}
	trackLocalsCopy := make(map[string]*webrtc.TrackLocalStaticRTP, len(r.rooms[joinUrl].trackLocals))
	for k, v := range r.rooms[joinUrl].trackLocals {
		trackLocalsCopy[k] = v
	}
	r.rooms[joinUrl].listLock.Unlock()

	for trackID := range trackLocalsCopy {
		if !existingSenders[trackID] {
			if _, err := pcState.peerConnection.AddTrack(trackLocalsCopy[trackID]); err != nil {
				log.Errorf("Failed to add track: %v", err)
			}

		}
	}
	if pcState.peerConnection.SignalingState() == webrtc.SignalingStateHaveLocalOffer {
		return
	}
	offer, err := pcState.peerConnection.CreateOffer(nil)
	if err != nil {
		log.Errorf("Failed to create offer: %v", err)
		return
	}

	if err = pcState.peerConnection.SetLocalDescription(offer); err != nil {
		log.Errorf("Failed to set local description: %v", err)
		return
	}

	offerString, err := json.Marshal(offer)
	if err != nil {
		log.Errorf("Failed to marshal offer: %v", err)
		return
	}

	if err = pcState.websocket.WriteJSON(&websocketStreamerMessage{
		Event: "offer",
		Data:  string(offerString),
	}); err != nil {
		log.Errorf("Failed to send offer: %v", err)
	}
}
func (r *Router) WebSocketChatHandler(ws *websocket.Conn) {
	joinUrl := ws.Query("join_url", "error")
	if joinUrl == "error" {
		return
	}

	c := &threadSafeWriter{Conn: ws, Mutex: sync.Mutex{}}
	defer c.Close()
	message := &websocketChatMessage{}
	for {
		_, raw, err := c.ReadMessage()
		if err != nil {
			return
		}

		log.Infof("Got message: %s", raw)

		if err := json.Unmarshal(raw, &message); err != nil {
			log.Errorf("Failed to unmarshal json to message: %v", err)
			return
		}

		switch message.Event {
		case "join":
			slog.Info(fmt.Sprintf("User %d joined in %s", message.From, message.ConferenceID))
			r.clientlock.Lock()
			if _, ok := r.clients[message.ConferenceID]; !ok {
				r.clients[message.ConferenceID] = &ChatRoom{
					listlock: sync.RWMutex{},
					conn:     make(map[*websocket.Conn]*threadSafeWriter),
				}
			}
			c.UserID = message.From

			r.clients[message.ConferenceID].conn[ws] = c
			r.clientlock.Unlock()
			if err != nil {
				slog.Error(err.Error())
			}
		case "message":
			err = r.service.Message.Send(&model.Message{
				ConferenceID: message.ConferenceID,
				SenderID:     message.From,
				Content:      message.Data,
				ContentType:  "text",
			})
			if err != nil {
				slog.Error(err.Error())
			}
			for _, el := range r.clients[message.ConferenceID].conn {
				err = el.Conn.WriteMessage(websocket.TextMessage, raw)
				if err != nil {
					slog.Error(err.Error())
				}
			}

		}
	}
}
func (r *Router) createPeerConnection() (*webrtc.PeerConnection, error) {
	m := webrtc.MediaEngine{}

	if err := m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:  webrtc.MimeTypeVP8,
			ClockRate: 90000,
			Channels:  0,
		},
		PayloadType: 96,
	}, webrtc.RTPCodecTypeVideo); err != nil {
		return nil, err
	}

	if err := m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:  webrtc.MimeTypeOpus,
			ClockRate: 48000,
			Channels:  2,
		},
		PayloadType: 111,
	}, webrtc.RTPCodecTypeAudio); err != nil {
		return nil, err
	}

	api := webrtc.NewAPI(webrtc.WithMediaEngine(&m))
	return api.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{webrtc.ICEServer{
			URLs: []string{"stun:stun.l.google.com:19302"},
		}},
		SDPSemantics: webrtc.SDPSemanticsUnifiedPlan,
	})
}
func (r *Router) WebSocketStreamerHandler(ws *websocket.Conn) {
	joinUrl := ws.Query("join_url", "error")
	if joinUrl == "error" {
		return
	}
	userID := ws.Query("user_id", "error")
	if userID == "error" {
		return
	}
	user_id, err := strconv.Atoi(userID)
	if err != nil {
		return
	}
	if _, ok := r.rooms[joinUrl]; !ok {
		r.rooms[joinUrl] = &Room{
			trackLocals:     make(map[string]*webrtc.TrackLocalStaticRTP),
			peerConnections: make([]peerConnectionState, 0),
		}
	}
	c := &threadSafeWriter{Conn: ws, Mutex: sync.Mutex{}, UserID: int64(user_id)}
	defer c.Close()
	peerConnection, err := r.createPeerConnection()
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
	r.rooms[joinUrl].listLock.Lock()
	r.rooms[joinUrl].peerConnections = append(r.rooms[joinUrl].peerConnections, peerConnectionState{peerConnection: peerConnection, websocket: c})
	r.rooms[joinUrl].listLock.Unlock()

	peerConnection.OnICECandidate(func(i *webrtc.ICECandidate) {
		if i == nil {
			return
		}
		candidateString, err := json.Marshal(i.ToJSON())
		if err != nil {
			log.Errorf("Failed to marshal candidate to json: %v", err)
			return
		}
		r.rooms[joinUrl].listLock.Lock()
		defer r.rooms[joinUrl].listLock.Unlock()
		if err = c.WriteJSON(&websocketStreamerMessage{
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
			r.signalPeerConnections(joinUrl)
		default:
		}
	})
	peerConnection.OnTrack(func(t *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		log.Infof("Got remote track: Kind=%s, ID=%s", t.Kind(), t.ID())

		trackLocal := r.addTrack(t, joinUrl)
		if trackLocal == nil {
			return
		}
		defer r.removeTrack(trackLocal, joinUrl)

		buf := rtpBufferPool.Get().([]byte)
		defer rtpBufferPool.Put(buf)
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

			if err = trackLocal.WriteRTP(rtpPkt); err != nil {
				log.Infof("Track writing stopped: %v", err)
				return
			}
		}
	})
	r.signalPeerConnections(joinUrl)

	peerConnection.OnICEConnectionStateChange(func(is webrtc.ICEConnectionState) {
		if is.String() == webrtc.PeerConnectionStateConnected.String() {
			r.signalPeerConnections(joinUrl)

		}
		log.Infof("ICE connection state changed: %s", is)
	})
	message := &websocketStreamerMessage{}
	for {
		_, raw, err := c.ReadMessage()
		if err != nil {
			return
		}

		if err := json.Unmarshal(raw, &message); err != nil {
			log.Errorf("Failed to unmarshal json to message: %v", err)
			return
		}

		switch message.Event {
		case "join":
			var joinMessage struct {
				UserID int64 `json:"user_id"`
			}
			if err := json.Unmarshal([]byte(message.Data), &joinMessage); err != nil {
				log.Errorf("Failed to unmarshal join message: %v", err)
				return
			}
			c.UserID = joinMessage.UserID

		case "candidate":
			candidate := webrtc.ICECandidateInit{}
			if err := json.Unmarshal([]byte(message.Data), &candidate); err != nil {
				log.Errorf("Failed to unmarshal json to candidate: %v", err)
				return
			}

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

			if err := peerConnection.SetRemoteDescription(answer); err != nil {
				log.Errorf("Failed to set remote description: %v", err)
				return
			}

		default:
			log.Errorf("unknown message: %+v", message)
		}
	}
}

func (r *Router) cleanupRooms() {
	for {
		time.Sleep(5 * time.Minute)
		r.roomslock.Lock()
		for url, room := range r.rooms {
			room.listLock.Lock()
			if len(room.peerConnections) == 0 && len(room.trackLocals) == 0 {
				delete(r.rooms, url)
				err := r.service.Conference.DeleteConference(url)
				if err != nil {
					slog.Error("error when trying to delete room from db", err)
				}
				log.Infof("Cleaned up unused room: %s", url)
			}
			room.listLock.Unlock()
		}
		r.roomslock.Unlock()
	}
}

func (r *Router) closeExistingConnections(userID int64) {
	r.roomslock.Lock()
	defer r.roomslock.Unlock()

	for _, room := range r.rooms {
		room.listLock.Lock()
		defer room.listLock.Unlock()

		for i := 0; i < len(room.peerConnections); i++ {
			if room.peerConnections[i].websocket.UserID == userID {
				if err := room.peerConnections[i].peerConnection.Close(); err != nil {
					log.Errorf("Failed to close peer connection: %v", err)
				}
				if err := room.peerConnections[i].websocket.Close(); err != nil {
					log.Errorf("Failed to close websocket: %v", err)
				}

				room.peerConnections = append(room.peerConnections[:i], room.peerConnections[i+1:]...)
				i--
			}
		}
	}

	r.clientlock.Lock()
	defer r.clientlock.Unlock()

	for _, chatRoom := range r.clients {
		chatRoom.listlock.Lock()
		for conn, writer := range chatRoom.conn {
			if writer.UserID == userID {
				if err := conn.Close(); err != nil {
					log.Errorf("Failed to close chat connection: %v", err)
				}
				delete(chatRoom.conn, conn)
			}
		}
		chatRoom.listlock.Unlock()
	}
}
