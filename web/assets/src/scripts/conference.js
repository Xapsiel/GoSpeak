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
                // "stun:global.stun.twilio.com:3478"
            ]
        },
        {
            urls: [
                // "turn:relay1.expressturn.com:3478?transport=udp",
                "turn:relay1.expressturn.com:3478?transport=tcp",
                // "turns:relay1.expressturn.com:5349?transport=tcp"
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
    }catch (error){
        console.error("Error accessing media devices:", error);
    }
}

function createPeerConnection(){
    state.peerConnection = new RTCPeerConnection(configuration);

    state.peerConnection.ontrack = function (event){

        let el = document.createElement(event.track.kind);
        el.srcObject = event.streams[0];
        el.autoplay =true;
        el.controls=false;

        document.getElementById("remoteVideos").appendChild(el);
        event.track.onmute = function (event){}
        event.track.onunmute = function (event){
            el.play();
        }
        event.streams[0].onremovetrack = ({track})=>{
            if (el.parentNode){
                el.parentNode.removeChild(el);
            }
        }
    }
    // document.getElementById('localVideo').srcObject = state.localStream;
    state.localStream.getTracks().forEach(track => state.peerConnection.addTrack(track,state.localStream))
    state.peerConnection.onicecandidate = e =>{
        if (!e.candidate){
            return;
        }
        state.ws.send(JSON.stringify({event: 'candidate', data: JSON.stringify(e.candidate)}));
    }
}

function  setupWebSocket(){
    const params = new URLSearchParams(window.location.search);
    const joinUrl = params.get('join_url');
    console.log(joinUrl)
    state.ws = new WebSocket(`ws://${domain}/ws?join_url=${joinUrl}`);
    state.ws.onopen = async () => {
        sendWSMessage({event: 'join', conference_id: conference.id, user_id: auth.user.user_id});
    };
    state.ws.onclose  = function(event){
        window.alert("websocket has closed");
    }
    state.ws.onerror=function (event){
        console.log("ERROR: "+event.data)
    }
    state.ws.onmessage = function (event){
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
                    state.ws.send(JSON.stringify({event: "answer", data: JSON.stringify(answer)}))
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

async function handleOffer(offer){
    await state.peerConnection.setRemoteDescription(offer);
    state.peerConnection.createAnswer().then(answer=>{
        state.peerConnection.setLocalDescription(answer)
        sendWSMessage({type: 'payload', answer});
    }
    );
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
    participants: new Map(), // участники конференции
};










document.addEventListener("DOMContentLoaded", () => {
    const urlParams = new URLSearchParams(window.location.search);
    const joinUrl = urlParams.get("join_url");

    const createSection = document.getElementById("createConference");
    const conferenceSection = document.getElementById("conferenceSection");

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
    await createPeerConnection();

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
