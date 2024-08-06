const userSignInDiv = document.getElementById("userSignIn");
const userID = document.getElementById("userID");

const userSignInBtn = document.getElementById("userIDSubmit");
userSignInBtn.addEventListener("click", getUser)

const mainScreenDiv = document.getElementById("mainScreen");

const writeDataBtn = document.getElementById("writeData");
writeDataBtn.addEventListener("click", writeData);

const userNameSpan = document.getElementById("userNameSpan");

const signOutLink = document.getElementById("signOut");
signOutLink.addEventListener("click", signOut);

const memoriesBtn = document.getElementById("memories");
memoriesBtn.addEventListener("click", () => {
    window.location.replace("/memories");
})

const recLogBtn = document.getElementById("recordLog");
recLogBtn.addEventListener("click", () => { captureAudio(); });

const recStopBtn = document.getElementById("recordStop");
recStopBtn.addEventListener("click", () => { processAudio(); });
const aud = document.getElementById("audio");

const logText = document.getElementById("logText");

const saveEditedText = document.getElementById("saveEditedText");
saveEditedText.addEventListener("click", saveEditedLogText);

const primaryHighlight = document.getElementById("primaryHighlight");
const secondaryHighlight = document.getElementById("secondaryHighlight");
const whoIWasWith = document.getElementById("whoIWasWith");

const summary = document.getElementById("summary");

// -------------------

let userName = "";
let sessionUserID = sessionStorage.getItem("userID") ?? "";
checkSignIn();

let mediaRecorder;
let audioStream;
let audioChunks = [];

allowRecording(true);

let customHighlights = JSON.parse(sessionStorage.getItem("customHighlights"));
populate(primaryHighlight, customHighlights);
populate(secondaryHighlight, customHighlights);

localStorage.setItem("lastAccessTime", `${new Date().toISOString()}`);

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

function allowRecording(s) {
    if (s == true) {
        recLogBtn.disabled = false;
        recLogBtn.innerText = "Record Log";
        recStopBtn.disabled = true;
        return
    }
    recLogBtn.disabled = true;
    recLogBtn.innerText = "recording";
    recStopBtn.disabled = false;
}

async function saveEditedLogText() {
    if (logText.value == "") { return }
    
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

async function playAudio() {
    const blob = new Blob(audioChunks, { type: 'audio/webm;codec=ogg' });
    const audioURL = URL.createObjectURL(blob);
    aud.src = audioURL;
    const ds = new Date().toISOString();
    aud.title = `log-${ds}.ogg`
    aud.play();
    sessionStorage.setItem("logDate", ds);
    recordLogEntry(ds, blob);
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

function checkSignIn() {
    if (sessionUserID == "") {
        showSignOnScreen()
        return
    }
    showMainScreen();
}

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

async function getUser() {
    const uid = userID.value;
    userName = await userIDExist(userID.value);
    console.log(`un: ${userName}`);
    if (userName == "") {
        alert(`Sorry: ${userID.value} was not found on the system.`);
        return;
    }
    sessionStorage.setItem("userID", userID.value);
    sessionUserID = userID.value;
    getHighlightSelections();
    showMainScreen();
}

async function userIDExist(userID) {
    try {
        const res = await fetch(`/userIDExist?userID=${userID}`);
        if (res.status != 200) {
            alert("Sorry, we're having some issue with the server at this time");
            return;
        }
        return await res.text();
    } catch (err) {
        console.error(err);
    }
}

function signOut() {
    sessionStorage.setItem("userID", "");
    sessionUserID = "";
    checkSignIn();

}

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
