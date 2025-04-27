import axios from 'https://cdn.jsdelivr.net/npm/axios/dist/esm/axios.min.js';


const state = {
    peerConnection : null,
    localStream: null,
    localStreamEl : null,
    remoteStreams: new Map(),
    streamWS: null,
    chatWS: null,
    conferenceId: null,
    userId: null
}
const configuration = {
    iceServers: [
        {
            urls: [
                "stun:stun.l.google.com:19302",
            ]
        },

    ]
};

async function initLocalStream(){
    try {
        state.localStream = await navigator.mediaDevices.getUserMedia({
            video: true,
            audio: true
        });
        if (state.localStreamEl!=null){
            state.localStreamEl.srcObject=state.localStream;
        }else{
            state.localStreamEl = document.createElement("video");
            state.localStreamEl.srcObject = state.localStream;
            state.localStreamEl.autoplay = true;
            state.localStreamEl.muted    = true;
            state.localStreamEl.playsInline = true;
            let card = createCard(state.localStreamEl)
            document.getElementById("remoteVideos").appendChild(card);
        }

        await state.localStreamEl.play();
    } catch (error) {
        console.error("Error accessing media devices:", error);
    }
}
function createCard(video){
    let card = document.createElement('div');
    card.className="col-12 col-md-6 col-lg-4 col-xl-3";
    let card2 =document.createElement('div')
    card2.className='card h-100 shadow-sm';
    let card_video_container = document.createElement('div');
    card_video_container.className='card-video-container ratio ratio-16x9';
    video.className='w-100 h-100'
    card_video_container.appendChild(video);
    card2.appendChild(card_video_container);
    card.appendChild(card2);
    return card
}
function createPeerConnection(){

    state.peerConnection = new RTCPeerConnection(configuration);

    state.peerConnection.ontrack = function (event){
        if (event.streams[0].id === state.localStream.id){
            return;
        }
        let el = document.createElement(event.track.kind);
        el.srcObject = event.streams[0];
        el.autoplay =true;
        el.controls=false;
        let card = createCard(el)
        if (event.track.kind === 'audio'){
            return
        }
        document.getElementById("remoteVideos").appendChild(card);
        event.track.onmute = function (event){}
        event.track.onunmute = function (event){
            el.play();
        }
        event.streams[0].onremovetrack = ({track})=>{
            if (card.parentNode){
                card.parentNode.removeChild(card);
            }
        }
    }
    state.localStream.getTracks().forEach(track => state.peerConnection.addTrack(track,state.localStream))
    state.peerConnection.onicecandidate = e =>{
        if (!e.candidate){
            return;
        }
        state.streamWS.send(JSON.stringify({event: 'candidate', data: JSON.stringify(e.candidate)}));
    }
}
async function setupStreamWebSocket(joinUrl){
    await initLocalStream();
    if (state.localStream){
        await createPeerConnection();
    }else{
        return
    }
    state.streamWS = new WebSocket(`ws://${domain}/ws/stream?join_url=${joinUrl}`);
    state.streamWS.onclose  = function(event){
        window.alert("websocket has closed");
    }
    state.streamWS.onerror=function (event){
        console.log("ERROR: "+event.data)
    }
    state.streamWS.onmessage = function (event){
        let msg = JSON.parse(event.data);
        if (!msg){
            return console.log("failed to parse msg")
        }
        switch (msg.event){
            case "answer":
                let answer = JSON.parse(msg.data);
                if (!answer){
                    return console.log("failed to parse answer");
                }
                state.peerConnection.setRemoteDescription(answer)
                return
            case "offer":
                let offer =JSON.parse(msg.data);
                if (!offer){
                    return console.log("failed to parse offer");
                }
                state.peerConnection.setRemoteDescription(offer);
                state.peerConnection.createAnswer().then(answer=>{
                    state.peerConnection.setLocalDescription(answer);
                    state.streamWS.send(JSON.stringify({event: "answer", data: JSON.stringify(answer)}))
                })
                return
            case "candidate":
                let candidate = JSON.parse(msg.data);
                if (!candidate){
                    return console.log("failed to parse candidate");
                }
                state.peerConnection.addIceCandidate(candidate);
        }
    }
}
function sendMessage() {
    const input = document.getElementById("chatInput");
    if (!input.value.trim() || !auth.user || !state.chatWS || state.chatWS.readyState !== WebSocket.OPEN) return;

    const messageData = {
        event: "message",
        conference_id: conference.id,
        from: auth.user.user_id,
        data: input.value,
    };

    state.chatWS.send(JSON.stringify(messageData));
    input.value = "";
}
function handleNewMessage(from, content ) {
    const chatMessages = document.getElementById("chatMessages");
    const message = document.createElement("div");
    message.classList.add("chat-message");
    message.textContent = `${from}: ${content}`;
    chatMessages.appendChild(message);
    chatMessages.scrollTop = chatMessages.scrollHeight;
}
async function setupChatWebSocket(joinUrl){
    state.chatWS = new WebSocket(`ws://${domain}/ws/chat?join_url=${joinUrl}`);
    state.chatWS.onopen = async ()=>{
        sendChatWSMessage({event:"join", conference_id: conference.id, from: auth.user.user_id,})
    }
    state.chatWS.onmessage=function (event){
        let msg = JSON.parse(event.data);
        if (!msg){
            return console.log("failed to parse msg")
        }
        switch (msg.event){
            case "join":
                console.log(`user ${msg.from} joined in ${msg.conference_id}`);
                return
            case "message":
                handleNewMessage(msg.from,msg.data);
                return
        }
    }

}
function  setupWebSocket(){
    const params = new URLSearchParams(window.location.search);
    const joinUrl = params.get('join_url');
    setupStreamWebSocket(joinUrl);

    setupChatWebSocket(joinUrl);

}

async function handleOffer(offer){
    await state.peerConnection.setRemoteDescription(offer);
    state.peerConnection.createAnswer().then(answer=>{
        state.peerConnection.setLocalDescription(answer)
        sendWSMessage({type: 'payload', answer});
    }
    );
}
function sendWSMessage(message){
    if (state.streamWS.readyState===WebSocket.OPEN){
        state.streamWS.send(JSON.stringify(message));
    }
}
function sendChatWSMessage(message){
    if (state.chatWS.readyState===WebSocket.OPEN){
        state.chatWS.send(JSON.stringify(message));
    }
}
function handleNewParticipant(userId) {
    console.log(`New participant: ${userId}`);
}

function handleLeftParticipant(userId) {
    if (state.remoteStreams.has(userId)) {
        document.getElementById(`remote-${userId}`).remove();
        state.remoteStreams.delete(userId);
    }
}


const elements = {
    createConferenceBtn: document.querySelector("#createConferenceButton"),
    createConferenceForm: document.querySelector("#createConferenceForm"),
};
const axiosInstance = axios.create({
    baseURL: `http://${domain}`,
});

const auth = {
    token: window.localStorage.getItem("jwtToken"),
    user: JSON.parse(window.localStorage.getItem("user") || null),
};

const conference = {
    id: 0,
    creator_id: 0,
    join_url: "",
    participants: new Map(),
};

// class RemoteVideoManager {
//     constructor() {
//         this.videoGrid = document.querySelector('.video-grid');
//         this.currentMode = 'grid';
//         this.setupModeToggle();
//     }
//
//     setupModeToggle() {
//         const toggleBtn = document.createElement('button');
//         toggleBtn.className = 'btn btn-icon';
//         toggleBtn.innerHTML = '<i class="fas fa-th-large"></i>';
//         toggleBtn.title = 'Переключить режим отображения';
//         toggleBtn.onclick = () => this.toggleDisplayMode();
//
//         document.querySelector('.conference-controls').appendChild(toggleBtn);
//     }
//
//     toggleDisplayMode() {
//         this.currentMode = this.currentMode === 'grid' ? 'focus' : 'grid';
//         this.videoGrid.classList.toggle('focus-mode');
//         this.videoGrid.classList.toggle('grid-mode');
//
//         if (this.currentMode === 'grid') {
//             const activeVideo = this.videoGrid.querySelector('.remote-video-container.active');
//             if (activeVideo) {
//                 activeVideo.classList.remove('active');
//             }
//         }
//     }
//
//     createRemoteVideoContainer(peerId, userName) {
//         const container = document.createElement('div');
//         container.className = 'remote-video-container';
//         container.dataset.peerId = peerId;
//
//         const video = document.createElement('video');
//         video.autoplay = true;
//         video.playsInline = true;
//
//         const nameLabel = document.createElement('div');
//         nameLabel.className = 'user-name';
//         nameLabel.textContent = userName;
//
//         container.appendChild(video);
//         container.appendChild(nameLabel);
//
//         container.onclick = () => {
//             if (this.currentMode === 'focus') {
//                 const activeVideo = this.videoGrid.querySelector('.remote-video-container.active');
//                 if (activeVideo) {
//                     activeVideo.classList.remove('active');
//                 }
//                 container.classList.add('active');
//             }
//         };
//
//         this.videoGrid.appendChild(container);
//         return container;
//     }
// }

document.addEventListener('DOMContentLoaded', () => {
    // window.remoteVideoManager = new RemoteVideoManager();

    const urlParams = new URLSearchParams(window.location.search);
    const joinUrl = urlParams.get("join_url");

    const createSection = document.getElementById("createConference");
    const conferenceSection = document.getElementById("conferenceSection");

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
                    initConference();
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
                const creator_id = auth.user.user_id;

                axiosInstance.post("/conference/create", { title, description, creator_id }, {
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
document.getElementById("sendMessage").addEventListener("click", sendMessage);
const toggleBtn = document.getElementById('toggleVideo');
toggleBtn.onclick = ()=>{
    const videoTrack = state.localStream.getVideoTracks()[0];
    if (videoTrack.enabled){
        videoTrack.enabled=false;
        videoTrack.stop();
        state.localStreamEl.srcObject = null;
    }else{
        initLocalStream();
        setupStreamWebSocket()
    }

}
async function initConference() {


    setupWebSocket();

}

function cleanup() {
    state.peerConnection?.close();
    state.streamWS?.close();
    state.localStream?.getTracks().forEach(track => track.stop());
    state.remoteStreams.forEach(stream => stream.getTracks().forEach(track => track.stop()));
}

window.addEventListener('beforeunload', () => {
});

const chatSection = document.querySelector('.chat-section');
const videoSection = document.querySelector('.video-section');

chatSection.addEventListener('mouseenter', () => {
    videoSection.classList.add('chat-open');
});

chatSection.addEventListener('mouseleave', () => {
    videoSection.classList.remove('chat-open');
});
