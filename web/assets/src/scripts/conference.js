import axios from 'https://cdn.jsdelivr.net/npm/axios/dist/esm/axios.min.js';

const elements = {
    createConferenceBtn: document.querySelector("#createConferenceButton"),
    createConferenceForm: document.querySelector("#createConferenceForm"),
};

let localPeerConnection;
let remotePeerConnection;
const axiosInstance = axios.create({
    baseURL: `http://${domain}`,
});
const auth = {
    token: window.localStorage.getItem("jwtToken"),
    user: JSON.parse(window.localStorage.getItem("user") || null),
};

const conference = {
    id: 0,
    creater_id: 0,
    join_url: "",
};

let ws = null;
let pendingCandidates = [];

function createPeerConnection() {
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
        localPeerConnection = new RTCPeerConnection(configuration);
        remotePeerConnection = new RTCPeerConnection(configuration);

        localPeerConnection.oniceconnectionstatechange = () => {
            console.log("Local ICE state:", localPeerConnection.iceConnectionState);
        };

        remotePeerConnection.oniceconnectionstatechange = () => {
            console.log("Remote ICE state:", remotePeerConnection.iceConnectionState);
        };

        localPeerConnection.onicecandidate = (event) => {
            if (event.candidate && ws && ws.readyState === WebSocket.OPEN) {
                ws.send(JSON.stringify({
                    type: "send_ice_candidate",
                    candidate: event.candidate,
                }));
            }
        };

        remotePeerConnection.onicecandidate = (event) => {
            if (event.candidate && ws && ws.readyState === WebSocket.OPEN) {
                ws.send(JSON.stringify({
                    type: "send_ice_candidate",
                    candidate: event.candidate,
                }));
            }
        };

        remotePeerConnection.ontrack = (event) => {
            const remoteVideo = document.getElementById("remoteVideo");
            if (remoteVideo.srcObject !== event.streams[0]) {
                remoteVideo.srcObject = event.streams[0];
            }
        };

    } catch (error) {
        console.error("PeerConnection creation failed:", error);
        throw error;
    }
}

function createOffer() {
    navigator.mediaDevices.getUserMedia({ video: true, audio: true })
        .then(localStream => {
            const localVideo = document.getElementById("localVideo");
            localVideo.srcObject = localStream;

            localStream.getTracks().forEach(track => {
                localPeerConnection.addTrack(track, localStream);
            });

            return localPeerConnection.createOffer();
        })
        .then(offer => localPeerConnection.setLocalDescription(offer))
        .then(() => {
            if (ws && ws.readyState === WebSocket.OPEN) {
                ws.send(JSON.stringify({
                    type: "send_offer",
                    offer: localPeerConnection.localDescription,
                }));
            }
        })
        .catch(error => console.error("Error in createOffer:", error));
}

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

function setupWebSocket() {
    if (!conference.id) {
        console.error("Ошибка: conference_id не установлен.");
        return;
    }

    ws = new WebSocket(`ws://${domain}/ws?=${conference.id}`);

    ws.onopen = () => {
        console.log("✅ WebSocket соединение установлено.");
        ws.send(JSON.stringify({
            type: "join_conference",
            user_id: auth.user?.user_id,
            creater_id: conference.creater_id,
            conference_id: conference.id,
        }));
    };

    ws.onmessage = (event) => {
        const messageData = JSON.parse(event.data);
        console.log("📩 Получено сообщение:", messageData);
        handleWebSocketMessage(messageData);
    };

    ws.onerror = (error) => console.error("❌ WebSocket ошибка:", error);
    ws.onclose = () => console.log("🔴 WebSocket соединение закрыто.");
}

function handleWebSocketMessage(data) {
    switch (data.response.type) {
        case "user_joined":
            handleUserJoined(data);
            break;
        case "user_left":
            handleUserLeft(data);
            break;
        case "new_message":
            handleNewMessage(data);
            break;
        case "receive_offer":
            handleReceiveOffer(data);
            break;
        case "receive_answer":
            handleReceiveAnswer(data);
            break;
        case "receive_ice_candidate":
            handleReceiveIceCandidate(data);
            break;
        default:
            console.warn("❓ Неизвестный тип сообщения:", data);
    }
}

function handleUserJoined(data) {
    console.log(`👤 Пользователь ${data} присоединился`);
}

function handleUserLeft(data) {
    console.log(`🚪 Пользователь ${data.data.user_id} покинул конференцию`);
}

function handleNewMessage(data) {
    const chatMessages = document.getElementById("chatMessages");
    const message = document.createElement("div");
    message.classList.add("chat-message");
    message.textContent = `${data.response.data.sender_id}: ${data.response.data.content}`;
    chatMessages.appendChild(message);
    chatMessages.scrollTop = chatMessages.scrollHeight;
}

function handleReceiveAnswer(data) {
    try {
        if (!data.response?.data) throw new Error("Invalid answer format");

        const answer = new RTCSessionDescription(data.response.data.answer);
        localPeerConnection.setRemoteDescription(answer)
            .then(() => console.log("Answer set successfully"))
            .catch(e => console.error("Error setting answer:", e));
    } catch (e) {
        console.error("Error handling answer:", e);
    }
}
var ts = (new Date()).getTime();

function handleReceiveIceCandidate(data) {
    try {
        if (!data.response?.data) throw new Error("Invalid ICE candidate format");

        const candidate = new RTCIceCandidate(data.response.data);

        if (!remotePeerConnection.remoteDescription?.type) {
            pendingCandidates.push(candidate);
            return;
        }

        remotePeerConnection.addIceCandidate(candidate)
            .then(() => console.log("ICE candidate added successfully"))
            .catch(e => console.error("Error adding ICE candidate:", e));
    } catch (e) {
        console.error("Error processing ICE candidate:", e);
    }
}

function handleReceiveOffer(data) {
    const offer = new RTCSessionDescription(data.response.data);

    remotePeerConnection.setRemoteDescription(offer)
        .then(() => remotePeerConnection.createAnswer())
        .then(answer => remotePeerConnection.setLocalDescription(answer))

        .then(() => {
            if (ws && ws.readyState === WebSocket.OPEN) {
                ws.send(JSON.stringify({
                    type: "send_answer",
                    answer: remotePeerConnection.localDescription,
                }));
            }
        })
        .then(() => {
            pendingCandidates.forEach(candidate => {
                remotePeerConnection.addIceCandidate(candidate)
                    .catch(e => console.error("Error adding pending candidate:", e));
            });
            pendingCandidates = [];
        })
        .catch(error => console.error("Error in handleReceiveOffer:", error));
}

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

// Event listeners
document.getElementById("sendMessage").addEventListener("click", sendMessage);
document.getElementById("chatInput").addEventListener("keypress", (event) => {
    if (event.key === "Enter") {
        event.preventDefault();
        sendMessage();
    }
});

document.getElementById("endCallButton").addEventListener("click", () => {
    if (ws) ws.close();
    alert("Звонок завершён.");
});

// Initialization
document.addEventListener("DOMContentLoaded", () => {
    const urlParams = new URLSearchParams(window.location.search);
    const joinUrl = urlParams.get("join_url");

    const createSection = document.getElementById("createConference");
    const conferenceSection = document.getElementById("conferenceSection");

    initializeUser().then(() => {
        console.log(auth.user, joinUrl);

        if (joinUrl) {
            axiosInstance.get(`/conference/join?join_url=${joinUrl}`, {
                headers: { Authorization: `Bearer ${auth.token}` }
            }).then(response => {
                if (response.data) {
                    conferenceSection?.classList.remove("d-none");
                    conference.id = response.data.conference_id;
                    conference.creater_id = response.data.creator_id;
                    conference.join_url = response.data.join_url;
                    console.log(conference);
                    setupWebSocket();
                    createPeerConnection();
                    createOffer();
                }
            }).catch(error => {
                console.error("Ошибка при присоединении:", error);
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
