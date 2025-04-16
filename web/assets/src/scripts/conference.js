import axios from 'https://cdn.jsdelivr.net/npm/axios/dist/esm/axios.min.js';

const elements = {
    createConferenceBtn: document.querySelector("#createConferenceButton"),
    createConferenceForm: document.querySelector("#createConferenceForm"),
};
const axiosInstance = axios.create({
    baseURL: `http://${domain}`,
});

// Храним все соединения (ключ: user_id, значение: RTCPeerConnection)
const peerConnections = new Map();
const localStreams = {}; // основной локальный поток
const remoteStreams = {}; // удалённые потоки по user_id

const auth = {
    token: window.localStorage.getItem("jwtToken"),
    user: JSON.parse(window.localStorage.getItem("user") || null),
};

const conference = {
    id: 0,
    creater_id: 0,
    join_url: "",
    participants: new Map(), // участники конференции
};

let ws = null;

// Обработчики для отправки сообщений в чате
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

    // Локальное добавление сообщения в чат
    const chatMessages = document.getElementById("chatMessages");
    const message = document.createElement("div");
    message.classList.add("chat-message");
    message.textContent = `${auth.user.user_id}: ${input.value}`;
    chatMessages.appendChild(message);
    chatMessages.scrollTop = chatMessages.scrollHeight;

    ws.send(JSON.stringify(messageData));
    input.value = "";
}

// Функция создания нового RTCPeerConnection с корректной конфигурацией
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

        // Обработчик изменения состояния ICE-соединения
        peerConnection.oniceconnectionstatechange = () => {
            console.log(`ICE state with ${targetUserId}:`, peerConnection.iceConnectionState);
            if (peerConnection.iceConnectionState === 'disconnected' ||
                peerConnection.iceConnectionState === 'failed') {
                cleanupConnection(targetUserId);
            }
        };

        // Отправка ICE кандидатов другому участнику
        peerConnection.onicecandidate = (event) => {
            if (event.candidate && ws && ws.readyState === WebSocket.OPEN) {
                ws.send(JSON.stringify({
                    type: "send_ice_candidate",
                    target_user_id: targetUserId,
                    candidate: event.candidate,
                }));
            }
        };

        // Обработка поступающих треков
        peerConnection.ontrack = (event) => {
            // Получаем контейнер для удалённых видео
            const remoteVideoContainer = document.getElementById("remoteVideos");
            let videoElement = document.getElementById(`remoteVideo-${targetUserId}`);

            // Если для данного участника видео еще не создано – создаем его и удалённый медиа-поток
            if (!videoElement) {
                videoElement = document.createElement('video');
                videoElement.id = `remoteVideo-${targetUserId}`;
                videoElement.autoplay = true;
                videoElement.playsInline = true;
                remoteVideoContainer.appendChild(videoElement);
            }
            if (!remoteStreams[targetUserId]) {
                remoteStreams[targetUserId] = new MediaStream();
                videoElement.srcObject = remoteStreams[targetUserId];
            }
            // Добавляем поступивший трек в соответствующий поток
            remoteStreams[targetUserId].addTrack(event.track);
        };

        // Сохраняем соединение
        peerConnections.set(targetUserId, peerConnection);
        return peerConnection;
    } catch (error) {
        console.error("PeerConnection creation failed:", error);
        throw error;
    }
}

// Очистка соединения, когда участник покидает конференцию
function cleanupConnection(userId) {
    const pc = peerConnections.get(userId);
    if (pc) {
        pc.close();
        peerConnections.delete(userId);
    }
    // Удаляем видео-элемент и очищаем поток
    const videoElement = document.getElementById(`remoteVideo-${userId}`);
    if (videoElement) {
        videoElement.parentNode.removeChild(videoElement);
    }
    delete remoteStreams[userId];
}

function handleNewMessage(data) {
    const chatMessages = document.getElementById("chatMessages");
    const message = document.createElement("div");
    message.classList.add("chat-message");
    message.textContent = `${data.sender_id}: ${data.content}`;
    chatMessages.appendChild(message);
    chatMessages.scrollTop = chatMessages.scrollHeight;
}

// Функция создания предложения (offer) для подключения к новому участнику
async function createOffer(targetUserId) {
    try {
        // Получаем локальный медиапоток, если он ещё не установлен
        if (!localStreams.main) {
            localStreams.main = await navigator.mediaDevices.getUserMedia({
                video: true,
                audio: true
            });
            const localVideo = document.getElementById("localVideo");
            if (localVideo) {
                localVideo.srcObject = localStreams.main;
                localVideo.muted = true;
            }
        }

        let peerConnection = peerConnections.get(targetUserId);
        if (!peerConnection) {
            peerConnection = createPeerConnection(targetUserId);
        }
        if (peerConnection.signalingState !== "stable") {
            peerConnection = createPeerConnection(targetUserId);
            return;
        }
        // Если треки еще не добавлены, добавляем их в соединение
        if (peerConnection.getSenders().length === 0) {
            localStreams.main.getTracks().forEach(track => {
                peerConnection.addTrack(track, localStreams.main);
            });
        }

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

// Обработка сообщений WebSocket
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
        // case "participants_list":
        //     handleParticipantsList(messageData);
        //     break;
        default:
            console.warn("Unknown message type:", data);
    }
}

// Обработка списка участников при входе в конференцию
// function handleParticipantsList(participants) {
//     participants.forEach(user_id => {
//         if (user_id !== auth.user.user_id) {
//             conference.participants.set(user_id, user_id);
//             createOffer(user_id);
//         }
//     });
// }
function handleUserJoined(data) {
    const userId = data.user_id;
    if (auth.user.user_id === userId) return;
    conference.participants.set(userId, data);
    createOffer(userId);
}

function handleUserLeft(data) {
    const userId = data.user_id;
    console.log(`User ${userId} left`);
    conference.participants.delete(userId);
    cleanupConnection(userId);
}

async function setupLocalCamera() {
    try {
        localStreams.main = await navigator.mediaDevices.getUserMedia({
            video: true,
            audio: true
        });
        document.getElementById("localVideo").srcObject = localStreams.main;
    } catch (error) {
        console.error("Camera error:", error);
    }
}
// При поступлении предложения (offer) от другого участника
async function handleReceiveOffer(data) {
    try {
        const { sender_id, offer } = data;

        let peerConnection = peerConnections.get(sender_id);
        if (!peerConnection) {
            peerConnection = createPeerConnection(sender_id);
        }

        await peerConnection.setRemoteDescription(new RTCSessionDescription(offer));

        if (!localStreams.main) {
            localStreams.main = await navigator.mediaDevices.getUserMedia({
                video: true,
                audio: true
            });
            const localVideo = document.getElementById("localVideo");
            if (localVideo) {
                localVideo.srcObject = localStreams.main;
                localVideo.muted = true;
            }
        }
        if (peerConnection.getSenders().length === 0) {
            localStreams.main.getTracks().forEach(track => {
                peerConnection.addTrack(track, localStreams.main);
            });
        }

        const answer = await peerConnection.createAnswer();
        await peerConnection.setLocalDescription(answer);  // ждём установку локального описания

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

// При поступлении ответа (answer) на наше предложение
async function handleReceiveAnswer(data) {
    try {
        const { sender_id, answer } = data;
        const peerConnection = peerConnections.get(sender_id);
        if (peerConnection) {
            await peerConnection.setRemoteDescription(new RTCSessionDescription(answer));
        }
    } catch (error) {
        console.error("Error in handleReceiveAnswer:", error);
    }
}

// Обработка ICE кандидатов
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

// Настройка WebSocket соединения. При подключении отправляем сообщения для входа и запроса списка участников
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
        // Очищаем все соединения при закрытии WS
        peerConnections.forEach((pc, userId) => {
            cleanupConnection(userId);
        });
    };
}

// Инициализация: раздел создания конференции или вход в конференцию
document.addEventListener("DOMContentLoaded", () => {
    const urlParams = new URLSearchParams(window.location.search);
    const joinUrl = urlParams.get("join_url");

    const createSection = document.getElementById("createConference");
    const conferenceSection = document.getElementById("conferenceSection");
    setupLocalCamera();

    initializeUser().then(() => {
        if (joinUrl) {
            // Запрос на вход в конференцию
            axiosInstance.get(`/conference/join?join_url=${joinUrl}`, {
                headers: { Authorization: `Bearer ${auth.token}` }
            }).then(response => {
                if (response.data) {
                    conferenceSection?.classList.remove("d-none");
                    conference.id = response.data.conference_id;
                    conference.creater_id = response.data.creater_id;
                    conference.join_url = response.data.join_url;
                    setupWebSocket();
                    // НЕ создаем лишнее соединение для себя
                    // createPeerConnection(auth.user.user_id);
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

// Закрытие соединений при уходе со страницы
window.addEventListener('beforeunload', () => {
    if (ws) ws.close();
    peerConnections.forEach((pc, userId) => {
        cleanupConnection(userId);
    });
});
