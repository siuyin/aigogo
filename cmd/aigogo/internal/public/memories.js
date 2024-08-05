const selectedSuggestion = document.getElementById("selected-suggestion");
selectedSuggestion.addEventListener("change", (ev) => {
    copySelectedSuggestionToUserPrompt(ev.target.value)
});

const userPrompt = document.getElementById("userPrompt");

const userSubmit = document.getElementById("userSubmit");
userSubmit.addEventListener("click", memGen);

const modelResponse = document.getElementById("modelResponse");


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

function updtPersonalLogRef() {
    const refs = document.querySelectorAll("a.popup");
    for (const r of refs) {
        r.addEventListener("click",(ev)=>{
            ev.preventDefault();
            alert(ev.target.innerHTML);
        });
        console.log(r.innerHTML);
    }
}