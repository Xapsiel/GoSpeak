import axios from 'https://cdn.jsdelivr.net/npm/axios/dist/esm/axios.min.js';


const state = {
    peerConnection : null,
    localStream: null,
    remoteStreams: new Map(),
    ws: null,
    conferenceId: null,
    userId: null
}
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

async function initLocalStream(){
    try{
        state.localStream= await navigator.mediaDevices.getUserMedia({
            video:true,
            audio:true
        });
        document.getElementById("localVideo").srcObject = state.localStream;
    }catch (error){
        console.error("Error accessing media devices:", error);
    }
}

function createPeerConnection(){
    const pc = new RTCPeerConnection(configuration);
    pc.onicecandidate= ({candidate})=>{
        if (candidate){
            sendWSMessage({type: "ice", candidate});
        }
    };
    pc.ontrack = ({streams,track})=>{
        const stream = streams[0];
        const userId = stream.id;
        if (!state.remoteStreams.has(userId)){
            const video =document.createElement("video");
            video.id = `remote-${userId}`;
            video.autoplay=true;
            video.playsInline=true;
            document.getElementById("remoteVideos").appendChild(video);
            state.remoteStreams.set(userId, stream);
        }
        document.getElementById(`remote-${userId}`).srcObject=stream;

    }
    state.localStream.getTracks().forEach(track=>{
        pc.addTrack(track, state.localStream);
    })
    return pc;
}
function  setupWebSocket(){
    state.ws = new WebSocket(`ws://${domain}/ws`);
    state.ws.onopen = ()=>{
        sendWSMessage({type: 'join', conference_id: state.conferenceId, user_id: auth.user.userId});
    };
    state.ws.onmessage = async ({data}) =>{
        const msg = JSON.parse(data);
        switch (msg.type){
            case "offer":
                await handleOffer(msg.offer);
                break;
            case "ice":
                await state.peerConnection.addIceCandidate(msg.candidate);
                break;
            case "newParticipant":
                handleNewParticipant(msg.user_id);
                break;
            case "leftParticipant":
                handleLeftParticipant(msg.user_id);
                break;
        }
    }
}
async function handleOffer(offer){
    await state.peerConnection.setRemoteDescription(offer);
    const answer = await state.peerConnection.createAnswer();
    await state.peerConnection.setLocalDescription(answer);
    sendWSMessage({type: 'answer', answer});
}
function sendWSMessage(message){
    if (state.ws.readyState===WebSocket.OPEN){
        state.ws.send(JSON.stringify(message));
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










document.addEventListener("DOMContentLoaded", () => {
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
async function initConference() {
    await initLocalStream();
    state.peerConnection = createPeerConnection();
    setupWebSocket();
}

function cleanup() {
    state.peerConnection?.close();
    state.ws?.close();
    state.localStream?.getTracks().forEach(track => track.stop());
    state.remoteStreams.forEach(stream => stream.getTracks().forEach(track => track.stop()));
}

window.addEventListener('beforeunload', () => {
});
