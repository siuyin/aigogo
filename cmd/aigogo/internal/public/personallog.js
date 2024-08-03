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
saveEditedText.addEventListener("click", saveEditedLogText);

// summary.innerHTML = `summary: ${logText.value} <p>tags: Primary=${primaryHighlight.value}, Secondary=${secondaryHighlight.value}, People=${selectedPeople(whoIWasWith.selectedOptions)}`;
async function saveEditedLogText() {
    const ds = sessionStorage.getItem("logDate");
    const latlng = sessionStorage.getItem("latlng");
    const neighborhood = sessionStorage.getItem("neighborhood");
    const url = `/data?editedlog=log-${encodeURIComponent(ds)}&userID=${sessionUserID}&latlng=${latlng}&neighborhood=${neighborhood}&primary=${primaryHighlight.value}&secondary=${secondaryHighlight.value}&people=${selectedPeople(whoIWasWith.selectedOptions)}`;
    const res = await fetch(url,
        {
            method: "POST",
            headers: { "Content-Type": "test/plain" },
            body: logText.value,
        });
    let resp = "";
    const dec = new TextDecoder("utf-8");
    for await (const chunk of res.body) {
        resp += dec.decode(chunk);
    }
    summary.innerText = resp;
}

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
        mediaRecorder = new MediaRecorder(audioStream, { mimeType: "audio/webm;codec=ogg", audioBitsPerSecond: 16000 });
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
    logText.innerHTML = "transcribing...";
}

const aud = document.getElementById("audio");
async function playAudio() {
    const blob = new Blob(audioChunks, { type: 'audio/webm;codec=ogg' });
    const audioURL = URL.createObjectURL(blob);
    aud.src = audioURL;
    const ds = new Date().toISOString();
    aud.title = `log-${ds}.ogg`
    aud.play();
    sessionStorage.setItem("logDate", ds);
    await recordLogEntry(ds, blob);
    updateQueryFunctions();
}

async function recordLogEntry(ds, blob) {
    const url = `/data?filename=log-${encodeURIComponent(ds)}&userID=${sessionUserID}`;
    const res = await fetch(url,
        {
            method: "POST",
            headers: { "Content-Type": "audio/webm;codec=ogg" },
            body: blob,
        });
    let resp = "";
    const dec = new TextDecoder("utf-8");
    for await (const chunk of res.body) {
        resp += dec.decode(chunk);
    }
    logText.innerText = resp;
}

localStorage.setItem("lastAccessTime", `${new Date().toISOString()}`);

function updateQueryFunctions() {
    queryFunctions.innerHTML = "TODO: query functions";
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

const writeDataBtn = document.getElementById("writeData");
writeDataBtn.addEventListener("click", writeData);
async function writeData() {
    const url = `/data?ter=pau`;
    const res = await fetch(url,
        {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ ID: "123456", User: "SiuYin", TimeStr: new Date().toISOString() }),
        });
    let rep = "";
    const dec = new TextDecoder("utf-8");
    for await (const chunk of res.body) {
        rep += dec.decode(chunk);
    }
    console.log(rep);
}
const userSignInDiv = document.getElementById("userSignIn");
const mainScreenDiv = document.getElementById("mainScreen");
const userNameSpan = document.getElementById("userNameSpan");

let userName = "";
let sessionUserID = sessionStorage.getItem("userID") ?? "";
function checkSignIn() {
    if (sessionUserID == "") {
        showSignOnScreen()
        return
    }
    showMainScreen();
}
checkSignIn();

function showSignOnScreen() {
    userSignInDiv.classList.remove("hide");
    mainScreenDiv.classList.add("hide");
}

function showMainScreen() {
    userSignInDiv.classList.add("hide");
    mainScreenDiv.classList.remove("hide");
    userName = "Kit Siew";
    userNameSpan.innerText = userName;
}

const userID = document.getElementById("userID");
const userSignInBtn = document.getElementById("userIDSubmit");
userSignInBtn.addEventListener("click", getUser)
function getUser() {
    console.log("return fake user id: 123456");
    sessionStorage.setItem("userID", "123456");
    sessionUserID = "123456";
    userName = "Kit Siew";
    showMainScreen();
}

const signOutLink = document.getElementById("signOut");
signOutLink.addEventListener("click", () => {
    sessionStorage.setItem("userID", "");
    sessionUserID = "";
    checkSignIn();
});