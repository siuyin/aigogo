const recLogBtn = document.getElementById("recordLog");
recLogBtn.addEventListener("click", () => { captureAudio(); });

const recStopBtn = document.getElementById("recordStop");
recStopBtn.addEventListener("click", () => { processAudio(); });
allowRecording(true);

function allowRecording(s) {
    if (s == true) {
        recLogBtn.disabled = false;
        recStopBtn.disabled = true;
        return
    }
    recLogBtn.disabled = true;
    recStopBtn.disabled = false;
}

const logText = document.getElementById("logText");

const saveEditedText = document.getElementById("saveEditedText");
saveEditedText.addEventListener("click", () => {
    summary.innerHTML = `summary: ${logText.value} <p>tags: Primary=${primaryHighlight.value}, Secondary=${secondaryHighlight.value}, People=${selectedPeople(whoIWasWith.selectedOptions)}`;
});

const summary = document.getElementById("summary");
const primaryHighlight = document.getElementById("primaryHighlight");
const secondaryHighlight = document.getElementById("secondaryHighlight");
const whoIWasWith = document.getElementById("whoIWasWith");

const queryFunctions = document.getElementById("queryFunctions");


function selectedPeople(peoplelist) {
    let ret = [];
    for (const p of peoplelist) {
        ret.push(p.innerText);
    }
    return ret.join("|");
}

let mediaRecorder;
let audioStream;
let audioChunks = [];
async function captureAudio() {
    try {
        audioChunks = [];
        allowRecording(false);
        audioStream = await navigator.mediaDevices.getUserMedia({ audio: true });
        mediaRecorder = new MediaRecorder(audioStream, { mimeType: "audio/webm;codecs=opus", audioBitsPerSecond: 16000 });
        mediaRecorder.ondataavailable = (event) => {
            audioChunks.push(event.data);
        }
        mediaRecorder.start();
    } catch (err) {
        allowRecording(true);
        console.error("problem capturing audio:", err);
        summary.innerHTML = `problem accessing microphone: ${err}`;
    }
}

async function processAudio() {
    mediaRecorder.stop();
    mediaRecorder.onstop = () => {
        allowRecording(true);
        const tracks = audioStream.getAudioTracks();
        tracks[0].stop();
        playAudio();
    }
    addFakeLogEntry();
}

const aud = document.getElementById("audio");
const logEntries = JSON.parse(localStorage.getItem("logEntries")) ?? [];
async function playAudio() {
    const blob = new Blob(audioChunks, { type: 'audio/webm;codecs=opus' });
    const audioURL = URL.createObjectURL(blob);
    aud.src = audioURL;
    const ds = new Date().toISOString();
    aud.title = `log-${ds}.webm`
    aud.play();
    logEntries.push({ title: `log-${ds}`, audio: `${await blobToDataURL(blob)}` });
    updateQueryFunctions();
}

localStorage.setItem("lastAccessTime", `${new Date().toISOString()}`);

function updateQueryFunctions() {
    queryFunctions.innerHTML = "";
    for (const e of logEntries) {
        queryFunctions.innerHTML += `<p>${e.title} <audio src="${e.audio}" controls></audio></p>`;
    }
    localStorage.setItem("logEntries", JSON.stringify(logEntries));
}

async function blobToDataURL(b) {
    try {
        const b64 = await new Promise(r => {
            const rd = new FileReader();
            rd.onloadend = () => r(rd.result);
            rd.readAsDataURL(b);
        });
        // return b64.slice(b64.indexOf(",") + 1);
        return b64;
    } catch (err) {
        console.error(err);
    }
}

function addFakeLogEntry() {
    logText.innerHTML = "Date: " + new Date().toISOString() +
        " Coordinates: " + sessionStorage.getItem("latlng") +
        " Neighborhood: " + sessionStorage.getItem("neighborhood") +
        "\nThe quick brown fox jumps over the lazy dog";
}