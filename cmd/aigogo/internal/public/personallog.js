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

const reviewLatestLogLink = document.getElementById("reviewLatestLog");
reviewLatestLogLink.addEventListener("click", () => {
    console.log("reviewing log");
    logText.innerText = "The quick brown fox jumps over the lazy dog.";
})

const logText = document.getElementById("logText");

const saveEditedText = document.getElementById("saveEditedText");
saveEditedText.addEventListener("click", () => {
    summary.innerHTML = `summary: ${logText.value} <p>tags: Primary=${primaryHighlight.value}, Secondary=${secondaryHighlight.value}, People=${selectedPeople(whoIWasWith.selectedOptions)}`;
});

const summary = document.getElementById("summary");
const primaryHighlight = document.getElementById("primaryHighlight");
const secondaryHighlight = document.getElementById("secondaryHighlight");
const whoIWasWith = document.getElementById("whoIWasWith");

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
            console.log(".");
        }
        mediaRecorder.start();
        console.log("capturing audio");
    } catch (err) {
        allowRecording(true);
        console.error("problem capturing audio:", err);
        summary.innerHTML = `problem accessing microphone: ${err}`;
    }
}

async function processAudio() {
    console.log(`len: ${audioChunks.length}`)
    console.log(audioChunks);
    mediaRecorder.stop();
    mediaRecorder.onstop = () => {
        allowRecording(true);
        const tracks = audioStream.getAudioTracks();
        tracks[0].stop();
        playAudio();
    }
}

const aud = document.getElementById("audio");
function playAudio() {
    const blob = new Blob(audioChunks, { type: 'audio/webm;codecs=opus' });
    const audioURL = URL.createObjectURL(blob);
    aud.src = audioURL;
    aud.play();
}