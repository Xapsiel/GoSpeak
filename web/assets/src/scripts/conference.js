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

const toggleVideo = document.getElementById('toggleVideo');
const toggleAudio = document.getElementById('toggleAudio');
const endCallButton = document.getElementById('endCallButton');
const copyButton = document.getElementById("CopyButton");
let isVideoEnabled = true;
let isAudioEnabled = true;

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
            state.localStreamEl.autoplay = false;
            state.localStreamEl.muted = true;
            state.localStreamEl.playsInline = true;
            let card = createCard(state.localStreamEl)
            document.getElementById("remoteVideos").appendChild(card);
        }

        // const startButton = document.createElement('button');
        // startButton.textContent = 'Начать конференцию';
        // startButton.className = 'start-conference-button';
        // startButton.onclick = async () => {
            try {
                await state.localStreamEl.play();
                // startButton.remove();
                
                if (state.peerConnection && state.localStream) {
                    state.localStream.getTracks().forEach(track => {
                        state.peerConnection.addTrack(track, state.localStream);
                    });
                }
            } catch (err) {
                console.error('Ошибка при воспроизведении:', err);
            }
        // };
        // document.getElementById("remoteVideos").appendChild(startButton);
        
        return true;
    } catch (error) {
        if (error.name === 'NotAllowedError' || error.name === 'PermissionDeniedError') {
            console.log('Пользователь отказал в доступе к медиаустройствам. Режим просмотра.');
            const placeholder = document.createElement('div');
            placeholder.className = 'video-placeholder';
            placeholder.innerHTML = `
                <div class="placeholder-content">
                    <svg class="placeholder-icon" viewBox="0 0 24 24">
                        <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 17.93c-3.95-.49-7-3.85-7-7.93 0-.62.08-1.21.21-1.79L9 15v1c0 1.1.9 2 2 2v1.93zm6.9-2.54c-.26-.81-1-1.39-1.9-1.39h-1v-3c0-.55-.45-1-1-1H8v-2h2c.55 0 1-.45 1-1V7h2c1.1 0 2-.9 2-2v-.41c2.93 1.19 5 4.06 5 7.41 0 2.08-.8 3.97-2.1 5.39z"/>
                    </svg>
                    <span>Вы в режиме просмотра</span>
                </div>
            `;
            let card = createCard(placeholder);
            document.getElementById("remoteVideos").appendChild(card);
            
            if (toggleVideo) toggleVideo.disabled = true;
            if (toggleAudio) toggleAudio.disabled = true;
            
            return false;
        }
        console.error('Ошибка при инициализации потока:', error);
        return false;
    }
}

function createCard(video){
    let card = document.createElement('div');
    card.className="card shadow-sm";
    card.style.padding='0px'
    card.addEventListener('click', () => {
        card.classList.toggle('expanded');
    });
    card.style.minWidth = '240px';
    let card_video_container = document.createElement('div');
    card_video_container.className='card-video-container';
    card_video_container.style.padding = '0px';
    video.className='w-100 h-100 object-fit-cover'

    card_video_container.appendChild(video);
    card.appendChild(card_video_container);
    return card
}
function createPeerConnection(){

    state.peerConnection = new RTCPeerConnection(configuration);

    state.peerConnection.ontrack = function (event) {
        if (state.localStream) {
            if (event.streams[0].id === state.localStream.id) {
                return;
            }
        }

        let el = document.createElement(event.track.kind);
        el.srcObject = event.streams[0];
        el.autoplay = false;
        el.controls = false;
        el.playsInline = true;
        
        let card = createCard(el);
        if (event.track.kind === 'audio'){
            return;
        }
        
        // const startButton = document.createElement('button');
        // startButton.textContent = 'Включить видео';
        // startButton.className = 'start-conference-button';
        // startButton.onclick = async () => {
        //     try {
        //         el.play();
        //         startButton.remove();
        //     } catch (err) {
        //         console.error('Ошибка при воспроизведении:', err);
        //     }
        // };
        // card.appendChild(startButton);
        
        document.getElementById("remoteVideos").appendChild(card);
        
        event.track.onmute = function (event){}
        event.track.onunmute = function (event){
            el.play().catch(err => {
                console.error('Ошибка при воспроизведении:', err);
            });
        }
        event.streams[0].onremovetrack = ({track}) => {
            if (card.parentNode){
                card.parentNode.removeChild(card);
            }
        }
    }
    
    if (state.localStream){
        state.localStream.getTracks().forEach(track => state.peerConnection.addTrack(track,state.localStream))
    }
    
    state.peerConnection.onicecandidate = e => {
        if (!e.candidate){
            return;
        }
        state.streamWS.send(JSON.stringify({event: 'candidate', data: JSON.stringify(e.candidate)}));
    }
}
async function setupStreamWebSocket(joinUrl){
    try{
        await initLocalStream();
    }catch (error){
        console.log(error)
    }
    await createPeerConnection();

    state.streamWS = new WebSocket(`ws://${domain}/ws/stream?join_url=${joinUrl}&user_id=${auth.user.user_id}`);
    state.streamWS.onclose  = function(event){
        // window.location.href = '/';
    }
    state.streamWS.onerror=function (event){
        // window.location.href = '/';
    }
    state.streamWS.onopen = function() {
        state.streamWS.send(JSON.stringify({
            event: "join",
            data: JSON.stringify({
                user_id: auth.user.user_id
            })
        }));
        
        
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
                state.peerConnection.setRemoteDescription(answer);
                break;

            case "offer":
                let offer = JSON.parse(msg.data);
                if (!offer){
                    return console.log("failed to parse offer");
                }
                state.peerConnection.setRemoteDescription(offer);
                state.peerConnection.createAnswer().then(answer => {
                    state.peerConnection.setLocalDescription(answer);
                    state.streamWS.send(JSON.stringify({
                        event: "answer",
                        data: JSON.stringify(answer)
                    }));
                });
                return;
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
    state.chatWS.onclose  = function(event){
        // window.location.href = '/';

    }
    state.chatWS.onerror = function(event){
        window.location.href = '/';

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
    title: "",
    participants: new Map(),
    description:"",
};

const urlParams = new URLSearchParams(window.location.search);

document.addEventListener('DOMContentLoaded', () => {

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
                    conference.description = response.data.description;
                    conference.title = response.data.title
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

    if (toggleVideo) {
        toggleVideo.addEventListener('click', toggleVideoStream);
    }
    
    if (toggleAudio) {
        toggleAudio.addEventListener('click', toggleAudioStream);
    }
    
    if (endCallButton) {
        endCallButton.addEventListener('click', endCall);
    }
    if (copyButton){
        copyButton.addEventListener("click", CopyToClipboard)
    }
});
async function CopyToClipboard() {
    try {
        let text = ""
        let join_url =urlParams.get("join_url")
        if (join_url){
            text += `Присоединяйтесь к видеоконференции: ${window.location.href}\n`
        }
        if (conference.title){
            text += `Тема: ${conference.title}\n`;
        }
        await navigator.clipboard.writeText(text);
    } catch (err) {
    }
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
document.getElementById("sendMessage").addEventListener("click", sendMessage);

function toggleVideoStream() {
    if (!state.localStream) return;

    const videoTracks = state.localStream.getVideoTracks();
    if (videoTracks.length > 0) {
        isVideoEnabled = !isVideoEnabled;
        videoTracks.forEach(track => {
            track.enabled = isVideoEnabled;
        });

        const icon = toggleVideo.querySelector('.control-icon');
        if (icon) {
            icon.innerHTML = isVideoEnabled
                ? '<path d="M17 10.5V7c0-.55-.45-1-1-1H4c-.55 0-1 .45-1 1v10c0 .55.45 1 1 1h12c.55 0 1-.45 1-1v-3.5l4 4v-11l-4 4z"/>'
                : '<path d="M21 6.5l-4 4V7c0-.55-.45-1-1-1H9.82L21 17.18V6.5zM3.27 2L2 3.27 4.73 6H4c-.55 0-1 .45-1 1v10c0 .55.45 1 1 1h12c.21 0 .39-.08.54-.18L19.73 20 21 18.73 3.27 2z"/>';
        }

        toggleVideo.classList.toggle('active', isVideoEnabled);
    }
}

function toggleAudioStream() {
    if (!state.localStream) return;

    const audioTracks = state.localStream.getAudioTracks();
    if (audioTracks.length > 0) {
        isAudioEnabled = !isAudioEnabled;
        audioTracks.forEach(track => {
            track.enabled = isAudioEnabled;
        });

        const icon = toggleAudio.querySelector('.control-icon');
        if (icon) {
            icon.innerHTML = isAudioEnabled
                ? '<path d="M12 14c1.66 0 3-1.34 3-3V5c0-1.66-1.34-3-3-3S9 3.34 9 5v6c0 1.66 1.34 3 3 3z"/><path d="M17 11c0 2.76-2.24 5-5 5s-5-2.24-5-5H5c0 3.53 2.61 6.43 6 6.92V21h2v-3.08c3.39-.49 6-3.39 6-6.92h-2z"/>'
                : '<path d="M16.5 12c0-1.77-1.02-3.29-2.5-4.03v2.21l2.45 2.45c.03-.2.05-.41.05-.63zm2.5 0c0 .94-.2 1.82-.54 2.64l1.51 1.51C20.63 14.91 21 13.5 21 12c0-4.28-2.99-7.86-7-8.77v2.06c2.89.86 5 3.54 5 6.71zM4.27 3L3 4.27 7.73 9H3v6h4l5 5v-6.73l4.25 4.25c-.67.52-1.42.93-2.25 1.18v2.06c1.38-.31 2.63-.95 3.69-1.81L19.73 21 21 19.73l-9-9L4.27 3zM12 4L9.91 6.09 12 8.18V4z"/>';
        }

        toggleAudio.classList.toggle('active', isAudioEnabled);
    }
}

function endCall() {
    if (state.localStream) {
        state.localStream.getTracks().forEach(track => track.stop());
    }

    if (state.peerConnection) {
        state.peerConnection.close();
    }

    if (state.streamWS) {
        state.streamWS.close();
    }

    if (state.chatWS) {
        state.chatWS.close();
    }

    window.location.href = '/';
}

async function initConference() {
    if (conference.description){
        document.getElementById("conference-description").textContent=conference.description;
    }else{
        document.getElementById("conference-description").remove();
    }
    if (conference.title){
        document.getElementById("conference-title" ).textContent=conference.title
    }else{
        document.getElementById("conference-title" ).remove();
    }
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
