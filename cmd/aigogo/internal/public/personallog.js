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

async function saveEditedLogText() {
    const ds = sessionStorage.getItem("logDate");
    const latlng = sessionStorage.getItem("latlng");
    const neighborhood = sessionStorage.getItem("neighborhood");
    const url = `/data?editedlog=log-${encodeURIComponent(ds)}&userID=${sessionUserID}&latlng=${latlng}&neighborhood=${neighborhood}&primary=${primaryHighlight.value}&secondary=${secondaryHighlight.value}&people=${whoIWasWith.value}`;
    const res = await fetch(url,
        {
            method: "POST",
            headers: { "Content-Type": "text/plain" },
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
    logText.value = "transcribing...";
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
    recordLogEntry(ds, blob);
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
    logText.value = resp;
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
    getHighlightSelections();
    showMainScreen();
}

const signOutLink = document.getElementById("signOut");
signOutLink.addEventListener("click", () => {
    sessionStorage.setItem("userID", "");
    sessionUserID = "";
    checkSignIn();
});

function populate(selectElement, opts) {
    selectElement.innerHTML = "";
    const placeholder = document.createElement("option");
    placeholder.setAttribute("disabled", "");
    placeholder.setAttribute("selected", "");
    placeholder.setAttribute("hidden", "");
    selectElement.appendChild(placeholder);

    let currentGroup;
    for (const o of opts) {
        const c0 = Array.from(o)[0];
        if (c0 == " ") {
            const opt = document.createElement("option");
            opt.innerText = o;
            opt.value = currentGroup + ":" + o.slice(1);
            selectElement.appendChild(opt);
            continue;
        }
        const opt = document.createElement("optgroup");
        opt.setAttribute("label", o)
        currentGroup = o;
        selectElement.appendChild(opt);
    }
}

let customHighlights = JSON.parse(sessionStorage.getItem("customHighlights"));
async function getHighlightSelections() {
    try {
        const url = `/getHighlightSelections?userID=${sessionUserID}`
        const res = await fetch(url);
        sessionStorage.setItem("customHighlights", await res.text())
        populate(primaryHighlight, customHighlights);
        populate(secondaryHighlight, customHighlights);
    } catch (err) {
        console.error(err);
    }
}
populate(primaryHighlight, customHighlights);
populate(secondaryHighlight, customHighlights);