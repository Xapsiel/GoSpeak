import axios from 'https://cdn.jsdelivr.net/npm/axios/dist/esm/axios.min.js';

const elements = {
    createConferenceBtn: document.querySelector("#createConferenceButton"),
    createConferenceForm: document.querySelector("#createConferenceForm"),
};
const axiosInstance = axios.create({
    baseURL: `http://${domain}`,
});
// Store all peer connections in a map (for mesh network)
const peerConnections = new Map(); // Key: user_id, Value: RTCPeerConnection
const localStreams = {}; // To store local streams for each connection
const auth = {
    token: window.localStorage.getItem("jwtToken"),
    user: JSON.parse(window.localStorage.getItem("user") || null),
};

const conference = {
    id: 0,
    creater_id: 0,
    join_url: "",
    participants: new Map(), // Track all participants in the conference
};

let ws = null;
document.getElementById("sendMessage").addEventListener("click", sendMessage);
document.getElementById("chatInput").addEventListener("keypress", (event) => {
    if (event.key === "Enter") {
        event.preventDefault();
        sendMessage();
    }
});
function sendMessage() {
    const input = document.getElementById("chatInput");
    if (!input.value.trim() || !auth.user || !ws || ws.readyState !== WebSocket.OPEN) return;

    const messageData = {
        type: "chat_message",
        conference_id: conference.id,
        sender_id: auth.user.user_id,
        content: input.value,
        sent_at: new Date().toISOString(),
    };

    const chatMessages = document.getElementById("chatMessages");
    const message = document.createElement("div");
    message.classList.add("chat-message");
    message.textContent = `${auth.user.user_id}: ${input.value}`;
    chatMessages.appendChild(message);
    chatMessages.scrollTop = chatMessages.scrollHeight;
    ws.send(JSON.stringify(messageData));
    input.value = "";
}
// Modified peer connection configuration
function createPeerConnection(targetUserId) {
    const configuration = {
        iceServers: [
            {
                urls: [
                    "stun:stun.l.google.com:19302",
                    "stun:global.stun.twilio.com:3478"
                ]
            },
            {
                urls: [
                    "turn:relay1.expressturn.com:3478?transport=udp",
                    "turn:relay1.expressturn.com:3478?transport=tcp",
                    "turns:relay1.expressturn.com:5349?transport=tcp"
                ],
                username: "ef47B9MOBBMFPVPIJO",
                credential: "9BZOLQ3r6Lxa9qTL"
            }
        ]
    };

    try {
        const peerConnection = new RTCPeerConnection(configuration);

        // Add event handlers
        peerConnection.oniceconnectionstatechange = () => {
            console.log(`ICE state with ${targetUserId}:`, peerConnection.iceConnectionState);
            if (peerConnection.iceConnectionState === 'disconnected' ||
                peerConnection.iceConnectionState === 'failed') {
                cleanupConnection(targetUserId);
            }
        };

        peerConnection.onicecandidate = (event) => {
            if (event.candidate && ws && ws.readyState === WebSocket.OPEN) {
                ws.send(JSON.stringify({
                    type: "send_ice_candidate",
                    target_user_id: targetUserId,
                    candidate: event.candidate,
                }));
            }
        };

        peerConnection.ontrack = (event) => {
            // For mesh, we need to handle multiple remote streams
            const remoteVideoContainer = document.getElementById("remoteVideos");
            let videoElement = document.getElementById(`remoteVideo-${targetUserId}`);

            if (!videoElement) {
                videoElement = document.createElement('video');
                videoElement.id = `remoteVideo-${targetUserId}`;
                videoElement.autoplay = true;
                videoElement.playsInline = true;
                remoteVideoContainer.appendChild(videoElement);
            }

            if (videoElement.srcObject !== event.streams[0]) {
                videoElement.srcObject = event.streams[0];
            }
        };

        // Store the connection
        peerConnections.set(targetUserId, peerConnection);
        return peerConnection;

    } catch (error) {
        console.error("PeerConnection creation failed:", error);
        throw error;
    }
}

// Clean up a connection when it's no longer needed
function cleanupConnection(userId) {
    const pc = peerConnections.get(userId);
    if (pc) {
        pc.close();
        peerConnections.delete(userId);

        // Remove the video element
        const videoElement = document.getElementById(`remoteVideo-${userId}`);
        if (videoElement) {
            videoElement.parentNode.removeChild(videoElement);
        }
    }
}
function handleNewMessage(data) {
    const chatMessages = document.getElementById("chatMessages");
    const message = document.createElement("div");
    message.classList.add("chat-message");
    message.textContent = `${data.sender_id}: ${data.content}`;
    chatMessages.appendChild(message);
    chatMessages.scrollTop = chatMessages.scrollHeight;
}
// Modified createOffer for mesh network
async function createOffer(targetUserId) {
    try {
        // Get local media if we haven't already
        if (!localStreams.main) {
            localStreams.main = await navigator.mediaDevices.getUserMedia({
                video: true,
                audio: true
            });

            const localVideo = document.getElementById("localVideo");
            if (localVideo) {
                localVideo.srcObject = localStreams.main;
            }
        }

        // Create or get existing peer connection
        let peerConnection = peerConnections.get(targetUserId);
        if (!peerConnection) {
            peerConnection = createPeerConnection(targetUserId);
        }

        // Add tracks if not already added
        if (peerConnection.getSenders().length === 0) {
            localStreams.main.getTracks().forEach(track => {
                peerConnection.addTrack(track, localStreams.main);
            });
        }

        // Create and send offer
        const offer = await peerConnection.createOffer();
        await peerConnection.setLocalDescription(offer);

        if (ws && ws.readyState === WebSocket.OPEN) {
            ws.send(JSON.stringify({
                type: "send_offer",
                target_user_id: targetUserId,
                offer: peerConnection.localDescription,
            }));
        }

    } catch (error) {
        console.error("Error in createOffer:", error);
    }
}

// Modified WebSocket message handler for mesh
function handleWebSocketMessage(data) {
    if (!data.response) return;

    const messageData = data.response.data || data.response;

    switch (data.response.type) {
        case "user_joined":
            handleUserJoined(messageData);
            break;
        case "user_left":
            handleUserLeft(messageData);
            break;
        case "new_message":
            handleNewMessage(messageData);
            break;
        case "receive_offer":
            handleReceiveOffer(messageData);
            break;
        case "receive_answer":
            handleReceiveAnswer(messageData);
            break;
        case "receive_ice_candidate":
            handleReceiveIceCandidate(messageData);
            break;
        case "participants_list":
            handleParticipantsList(messageData);
            break;
        default:
            console.warn("Unknown message type:", data);
    }
}

// Handle the list of participants when joining
function handleParticipantsList(participants) {
    participants.forEach(participant => {
        if (participant.user_id !== auth.user.user_id) {
            conference.participants.set(participant.user_id, participant);
            createOffer(participant.user_id); // Initiate connection to each participant
        }
    });
}

// Modified user joined handler
function handleUserJoined(data) {
    const userId = data.user_id;
    if (auth.user.user_id===userId){
        return
    }
    console.log(`User ${userId} joined`);
    conference.participants.set(userId, data);
    createOffer(userId);
}

// Modified user left handler
function handleUserLeft(data) {
    const userId = data.user_id;
    console.log(`User ${userId} left`);
    conference.participants.delete(userId);
    cleanupConnection(userId);
}
async function setupLocalCamera() {
    try {
        if (!localStreams.main) {
            localStreams.main = await navigator.mediaDevices.getUserMedia({
                video: true,
                audio: true
            });

            const localVideo = document.getElementById("localVideo");
            if (localVideo) {
                localVideo.srcObject = localStreams.main;
                localVideo.muted = true; // Добавляем muted для локального видео
                console.log("Local camera stream set up successfully");
            }
        }
    } catch (error) {
        console.error("Error setting up local camera:", error);
        // Можно добавить обработку ошибки, например, показать сообщение пользователю
        alert("Could not access camera/microphone. Please check permissions.");
    }
}
// Modified offer handler for mesh
async function handleReceiveOffer(data) {
    try {
        const { sender_id, offer } = data;

        // Create or get existing peer connection
        let peerConnection = peerConnections.get(sender_id);
        if (!peerConnection) {
            peerConnection = createPeerConnection(sender_id);
        }

        // Set remote description
        await peerConnection.setRemoteDescription(new RTCSessionDescription(offer));

        // Get local media if we haven't already
        if (!localStreams.main) {
            localStreams.main = await navigator.mediaDevices.getUserMedia({
                video: true,
                audio: true
            });

            const localVideo = document.getElementById("localVideo");
            if (localVideo) {
                localVideo.srcObject = localStreams.main;
            }
        }

        // Add tracks if not already added
        if (peerConnection.getSenders().length === 0) {
            localStreams.main.getTracks().forEach(track => {
                peerConnection.addTrack(track, localStreams.main);
            });
        }

        // Create and send answer
        const answer = await peerConnection.createAnswer();
        peerConnection.setLocalDescription(answer);

        if (ws && ws.readyState === WebSocket.OPEN) {
            ws.send(JSON.stringify({
                type: "send_answer",
                target_user_id: sender_id,
                answer: peerConnection.localDescription,
            }));
        }

    } catch (error) {
        console.error("Error in handleReceiveOffer:", error);
    }
}

// Modified answer handler for mesh
async function handleReceiveAnswer(data) {
    try {
        const { sender_id, answer } = data;
        const peerConnection = peerConnections.get(sender_id);

        if (peerConnection) {
             peerConnection.setRemoteDescription(new RTCSessionDescription(answer));
        }
    } catch (error) {
        console.error("Error in handleReceiveAnswer:", error);
    }
}

// Modified ICE candidate handler for mesh
async function handleReceiveIceCandidate(data) {
    try {
        const { sender_id, candidate } = data;
        const peerConnection = peerConnections.get(sender_id);

        if (peerConnection) {
            await peerConnection.addIceCandidate(new RTCIceCandidate(candidate));
        }
    } catch (error) {
        console.error("Error in handleReceiveIceCandidate:", error);
    }
}

// Modified WebSocket setup to request participants list
function setupWebSocket() {
    if (!conference.id) {
        console.error("Error: conference_id not set.");
        return;
    }

    ws = new WebSocket(`ws://${domain}/ws?conference_id=${conference.id}`);

    ws.onopen = () => {
        console.log("WebSocket connection established.");
        ws.send(JSON.stringify({
            type: "join_conference",
            user_id: auth.user?.user_id,
            creater_id: conference.creater_id,
            conference_id: conference.id,
        }));

        // Request list of current participants
        ws.send(JSON.stringify({
            type: "request_participants",
            conference_id: conference.id,
        }));
    };

    ws.onmessage = (event) => {
        const messageData = JSON.parse(event.data);
        console.log("Received message:", messageData);
        handleWebSocketMessage(messageData);
    };

    ws.onerror = (error) => console.error("WebSocket error:", error);
    ws.onclose = () => {
        console.log("WebSocket connection closed.");
        // Clean up all connections when WS closes
        peerConnections.forEach((pc, userId) => {
            cleanupConnection(userId);
        });
    };
}

// Modified initialization
document.addEventListener("DOMContentLoaded", () => {
    const urlParams = new URLSearchParams(window.location.search);
    const joinUrl = urlParams.get("join_url");

    const createSection = document.getElementById("createConference");
    const conferenceSection = document.getElementById("conferenceSection");
    setupLocalCamera();

    initializeUser().then(() => {
        if (joinUrl) {
            axiosInstance.get(`/conference/join?join_url=${joinUrl}`, {
                headers: { Authorization: `Bearer ${auth.token}` }
            }).then(response => {
                if (response.data) {
                    conferenceSection?.classList.remove("d-none");
                    conference.id = response.data.conference_id;
                    conference.creater_id = response.data.creater_id;
                    conference.join_url = response.data.join_url;
                    setupWebSocket();
                    createPeerConnection(auth.user.user_id)
                }
            }).catch(error => {
                console.error("Join error:", error);
            });
        } else {
            createSection?.classList.remove("d-none");
        }
    });
    document.getElementById('createConferenceButton').addEventListener('click', () => {
        if (elements.createConferenceForm) {
            elements.createConferenceForm.addEventListener("submit", (event) => {
                event.preventDefault();
                const title = elements.createConferenceForm.querySelector("#title").value;
                const description = elements.createConferenceForm.querySelector("#description").value;
                const creater_id = auth.user.user_id;

                axiosInstance.post("/conference/create", { title, description, creater_id }, {
                    headers: { Authorization: `Bearer ${auth.token}` }
                }).then(response => {
                    if (response.data?.join_url) {
                        const newUrl = new URL(window.location.href);
                        newUrl.searchParams.set('join_url', response.data.join_url);
                        window.location.href = newUrl.toString();
                    } else {
                        console.error("Ошибка: join_url отсутствует в ответе сервера");
                    }
                }).catch(error => {
                    console.error(error);
                });
            });
        }
    });
});
function getUser() {
    return axiosInstance.get("/user/", {
        headers: { "Authorization": `Bearer ${auth.token}` },
    }).then(response => response.data)
        .catch(() => null);
}

function initializeUser() {
    if (auth.token) {
        return getUser().then(user => {
            if (user) {
                auth.user = user;
                window.localStorage.setItem("user", JSON.stringify(user));
            }
        });
    }
    return Promise.resolve();
}
window.addEventListener('beforeunload', () => {
    if (ws) ws.close();
    peerConnections.forEach((pc, userId) => {
        cleanupConnection(userId);
    });
});

