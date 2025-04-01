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
            if (event.candidate) {
                ws.send(JSON.stringify({
                    type: "send_ice_candidate",
                    candidate: event.candidate,
                }));
            }
        };

        remotePeerConnection.onicecandidate = (event) => {
            if (event.candidate) {
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


async function createOffer() {

    const localStream = await navigator.mediaDevices.getUserMedia({ video: true, audio: true });
    const localVideo = document.getElementById("localVideo");
    localVideo.srcObject = localStream;

    localStream.getTracks().forEach((track) => {
        localPeerConnection.addTrack(track, localStream);
    });

    const offer = await localPeerConnection.createOffer();
    await localPeerConnection.setLocalDescription(offer);

    ws.send(JSON.stringify({
        type: "send_offer",
        offer: offer,
    }));
}
async function getUser() {
    try {
        const response = await axiosInstance.get("/user/", {
            headers: { "Authorization": `Bearer ${auth.token}` },
        });
        return response.data;
    } catch (error) {
        return null;
    }
}

async function initializeUser() {
    if (auth.token) {
        const user = await getUser();
        if (user) {
            auth.user = user;
            window.localStorage.setItem("user", JSON.stringify(user));
        }
    }
}


function setupWebSocket() {
    if (!conference.id) {
        console.error("Ошибка: conference_id не установлен.");
        return;
    }

    ws = new WebSocket(`ws://${domain}/ws?=${conference.id}`);

    ws.onopen = () => {
        console.log("✅ WebSocket соединение установлено.");
        ws.send(JSON.stringify({ type: "join_conference",
            user_id: auth.user?.user_id,
            creater_id: conference.creater_id,
            conference_id: conference.id,
        } ));
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
    console.log(data)
    switch (data.response.type) {
        case "user_joined":
            handleUserJoined(data);
            break; //обработал
        case "user_left":
            handleUserLeft(data);
            break; //обработано
        case "new_message":
            handleNewMessage(data);
            break;//делаю
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


async function handleReceiveAnswer(data) {
    try {
        console.log("Received answer:", data);

        if (!data.response || !data.response.data) {
            throw new Error("Invalid answer format");
        }

        const answer = new RTCSessionDescription(data.response.data);
        console.log("Parsed answer:", answer);

        await localPeerConnection.setRemoteDescription(answer);
        console.log("Answer set successfully");
    } catch (e) {
        console.error("Error handling answer:", e);
    }
}

let pendingCandidates = [];

async function handleReceiveIceCandidate(data) {
    try {
        console.log("Received ICE candidate:", data);

        if (!data.response || !data.response.data) {
            throw new Error("Invalid ICE candidate format");
        }

        const candidate = new RTCIceCandidate(data.response.data);
        console.log("Parsed ICE candidate:", candidate);

        if (!remotePeerConnection.remoteDescription || !remotePeerConnection.remoteDescription.type) {
            console.warn("Remote description not set. Storing ICE candidate...");
            pendingCandidates.push(candidate);
            return;
        }

        await remotePeerConnection.addIceCandidate(candidate);
        console.log("ICE candidate added successfully");
    } catch (e) {
        console.error("Error adding ICE candidate:", e);
    }
}

async function handleReceiveOffer(data) {
    const offer = new RTCSessionDescription(data.response.data);
    await remotePeerConnection.setRemoteDescription(offer);

    console.log("Applying pending ICE candidates...");
    while (pendingCandidates.length) {
        await remotePeerConnection.addIceCandidate(pendingCandidates.shift());
    }

    const answer = await remotePeerConnection.createAnswer();
    await remotePeerConnection.setLocalDescription(answer);

    ws.send(JSON.stringify({
        type: "send_answer",
        answer: answer,
    }));
}


function sendMessage() {
    const input = document.getElementById("chatInput");
    if (!input.value.trim() || !auth.user || !ws || ws.readyState !== WebSocket.OPEN) {
        return;
    }

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

document.addEventListener("DOMContentLoaded", async () => {
    const urlParams = new URLSearchParams(window.location.search);
    const joinUrl = urlParams.get("join_url");

    const createSection = document.getElementById("createConference");
    const conferenceSection = document.getElementById("conferenceSection");
    await initializeUser()
    console.log(auth.user,joinUrl)
    if (joinUrl) {
        try {
            const response = await axiosInstance.get(
                `/conference/join?join_url=${joinUrl}`
                ,
                { headers: { Authorization: `Bearer ${auth.token}` } }
            );

            if (response.data) {
                conferenceSection?.classList.remove("d-none");
                conference.id = response.data.conference_id;
                conference.creater_id = response.data.creator_id;
                conference.join_url = response.data.join_url;
                console.log(conference)
                setupWebSocket();
                createPeerConnection();
                createOffer()
            }
        } catch (error) {
            console.error("Ошибка при присоединении:", error);
        }
    } else {
        createSection?.classList.remove("d-none");
    }

    document.getElementById('createConferenceButton').addEventListener('click', () => {
        if (elements.createConferenceForm) {
            console.log(1);
            elements.createConferenceForm.addEventListener("submit", async (event) => {
                event.preventDefault();
                const title = elements.createConferenceForm.querySelector("#title").value;
                const description = elements.createConferenceForm.querySelector("#description").value;
                const creater_id = auth.user.user_id;
                console.log(title, description, creater_id);
                try {
                    const response = await axiosInstance.post("/conference/create", { title, description, creater_id }, {
                        headers: {
                            Authorization: `Bearer ${auth.token}`
                        }
                    });

                    if (response.data && response.data.join_url) {
                        const joinUrl = response.data.join_url;

                        const newUrl = new URL(window.location.href);
                        newUrl.searchParams.set('join_url', joinUrl);

                        window.location.href = newUrl.toString();
                    } else {
                        console.error("Ошибка: join_url отсутствует в ответе сервера");
                    }

                } catch (error) {
                    console.error(error);
                }
            });
        }
    });
});

