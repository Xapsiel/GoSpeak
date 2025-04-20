package service

import (
	"encoding/json"
	"log"

	"GoSpeak/internal/model"
	"GoSpeak/internal/repository"

	"github.com/gofiber/websocket/v2"
	"github.com/pion/webrtc/v3"
)

type WebRTCService struct {
	repo repository.Repository
}

func NewWebRTCService(repo repository.Repository) *WebRTCService {
	return &WebRTCService{repo: repo}
}

func (w WebRTCService) CreatePeerConnection(room *model.Room, conn *websocket.Conn) *webrtc.PeerConnection {
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
			{URLs: []string{"turn:relay1.expressturn.com:3478?transport=tcp"},
				Username:       "ef47B9MOBBMFPVPIJO",
				Credential:     "9BZOLQ3r6Lxa9qTL",
				CredentialType: webrtc.ICECredentialTypePassword,
			},
		},
	}
	pc, err := webrtc.NewPeerConnection(config)
	if err != nil {
		panic(err)
	}
	pc.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		log.Printf("ICE Connection State has changed %s \n", connectionState.String())
	})
	pc.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			panic("ICECandidate is nil")
		}
		conn.WriteJSON(model.WebSocketMessage{
			Type:    "ice-candidate",
			Payload: json.RawMessage(candidate.ToJSON().Candidate),
		})
	})
	pc.OnTrack(func(remoteTrack *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		localTrack, _ := webrtc.NewTrackLocalStaticRTP(
			remoteTrack.Codec().RTPCodecCapability,
			remoteTrack.ID(),
			remoteTrack.StreamID())
		room.Mu.RLock()
		defer room.Mu.RUnlock()
		for _, p := range room.Participant {
			if p.Pc != pc && !p.IsPublisher {
				p.Pc.AddTrack(localTrack)
			}
		}
		go w.RelayTrack(remoteTrack, localTrack)
	})
	return pc
}

func (w *WebRTCService) RelayTrack(remote *webrtc.TrackRemote, local *webrtc.TrackLocalStaticRTP) {
	for {
		pkt, _, err := remote.ReadRTP()
		if err != nil {
			panic(err)
		}
		if err := local.WriteRTP(pkt); err != nil {
			panic(err)
		}

	}
}

func (w *WebRTCService) ReceiveOffer(peer *model.Peer, offer webrtc.SessionDescription) error {
	err := peer.Pc.SetRemoteDescription(offer)
	if err != nil {
		return err
	}
	answer, err := peer.Pc.CreateAnswer(nil)
	if err != nil {
		return err
	}
	err = peer.Pc.SetLocalDescription(answer)
	if err != nil {
		return err
	}
	ans, err := json.Marshal(answer)
	if err != nil {
		return err
	}
	err = peer.Conn.WriteJSON(model.WebSocketMessage{
		Type:    "answer",
		Payload: ans,
	})
	if err != nil {
		return err
	}
	return nil
}

func (w WebRTCService) ReceiveICECandidate(peer *model.Peer, candidate webrtc.ICECandidateInit) error {
	return peer.Pc.AddICECandidate(candidate)
}
