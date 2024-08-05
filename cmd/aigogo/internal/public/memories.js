const selectedSuggestion = document.getElementById("selected-suggestion");
selectedSuggestion.addEventListener("change", (ev) => {
    copySelectedSuggestionToUserPrompt(ev.target.value)
});

const userPrompt = document.getElementById("userPrompt");

const userSubmit = document.getElementById("userSubmit");
userSubmit.addEventListener("click", memGen);

const modelResponse = document.getElementById("modelResponse");

const selectedLogEntry = document.getElementById("selectedLogEntry");

// ------------------------------

import { marked } from "https://cdn.jsdelivr.net/npm/marked/lib/marked.esm.js";

let sessionUserID = sessionStorage.getItem("userID") ?? "";
if (sessionUserID == "") {
    window.location.replace("/personallog");
}

function copySelectedSuggestionToUserPrompt(prompt) {
    userPrompt.value = prompt;
}

async function memGen() {
    const url = `/memgen?userPrompt=${userPrompt.value}&userID=${sessionUserID}`;

    modelResponse.innerHTML = "working ... give me a few seconds ..."
    try {
        modelResponse.innerText = "";
        await streamToElement(modelResponse, url);
        updtPersonalLogRef();
    } catch (err) {
        console.error(err.message);
    }
}

function updtPersonalLogRef() {
    selectedLogEntry.innerHTML = "";
    const refs = document.querySelectorAll("a.popup");
    for (const r of refs) {
        r.addEventListener("click", (ev) => {
            ev.preventDefault();
            fetchPersonalLogDetails(ev.target.innerText);
        });
        console.log(r.innerHTML);
    }
}

async function fetchPersonalLogDetails(logBasename) {
    try {
        const url = `/ref?log=${logBasename}&userID=${sessionUserID}`;
        const res = await fetch(url);
        if (res.status != 200) {
            selectedLogEntry.innerHTML("could not fetch selected log entry");
            return;
        }
        const logDet = await res.json();
        selectedLogEntry.innerHTML = `<div>${logDet.Date}:
        <p><span class="heading">summary:</span> ${logDet.Summary}</p >
            <p><span class="heading">transcript:</span> ${logDet.Transcript}</p>
        <p><audio controls src="data:audio/ogg;base64,${logDet.Audio}"></audio>
        </div > `;
    } catch (err) {

    }
}

async function streamToElement(el, url) {
    const res = await fetch(url);
    let tmp = "";
    el.innerHTML = "";
    const dec = new TextDecoder("utf-8");
    for await (const chunk of res.body) {
        el.innerHTML += (dec.decode(chunk));
        tmp += (dec.decode(chunk));
    }
    el.innerHTML = marked.parse(tmp);
}
