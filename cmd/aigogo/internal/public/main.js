const geoCoords = document.getElementById("geoCoords");
if (!("geolocation" in navigator)) {
    console.log("geo location API not available");
}

navigator.geolocation.getCurrentPosition((position) => {
    geoCoords.innerText = `I see you are roughly located at coordinates: (${position.coords.latitude}, ${position.coords.longitude})`;
    updateNeighborhood(position.coords.latitude, position.coords.longitude);
    weatherForecast(position.coords.latitude, position.coords.longitude);
    sessionStorage.setItem("latlng", `${position.coords.latitude},${position.coords.longitude}`);
});

async function updateNeighborhood(lat, lng) {
    const url = `/loc?latlng=${lat},${lng}`;
    try {
        const resp = await fetch(url);
        if (!resp.ok) { throw new Error(`response status: ${resp.status}`) }
        const respTxt = await resp.text();
        geoCoords.innerText = `I see you are in the neighbourhood of ${respTxt}`;
        sessionStorage.setItem("neighborhood", respTxt);
    } catch (err) {
        console.error(err.message);
    }
}

async function weatherForecast(lat, lng) {
    const url = `https://api.open-meteo.com/v1/forecast?latitude=${lat}&longitude=${lng}&hourly=temperature_2m,apparent_temperature,precipitation_probability&forecast_hours=6&timezone=auto`;
    try {
        const resp = await fetch(url);
        if (!resp.ok) { throw new Error(`response status: ${resp.status}`) }
        const respJSON = await resp.text();
        sessionStorage.setItem("weatherForecastJSON", respJSON);
    } catch (err) {
        console.error(err.message);
    }
}

const currentTime = document.getElementById("currentTime");
currentTime.innerText = `Good ${getDayPart(new Date)}`;

const userPrompt = document.getElementById("userPrompt");
const userSubmit = document.getElementById("userSubmit");
userSubmit.addEventListener("click", retrieveDocsForAugmentation);

// embeddingResponse is the main RAG response
const embeddingResponse = document.getElementById("embeddingResponse");
// modelResponse is the LLM model response to "meaning of life"
const modelResponse = document.getElementById("modelResponse");

import { marked } from "https://cdn.jsdelivr.net/npm/marked/lib/marked.esm.js";
async function retrieveDocsForAugmentation() {
    const loc = encodeURIComponent(sessionStorage.getItem("neighborhood"));
    let usrQry = encodeURIComponent(userPrompt.value);
    let ctx = encodeURIComponent(sessionStorage.getItem("context"));
    let weather = encodeURIComponent(sessionStorage.getItem("weatherForecastJSON"));
    const url = `/retr?userPrompt=${usrQry}&loc=${loc}&latlng=${sessionStorage.getItem("latlng")}&ctx=${ctx}&weather=${weather}`;

    embeddingResponse.innerHTML = "working ... give me a few seconds ..."
    try {
        modelResponse.innerText = "";
        await streamToElement(embeddingResponse, url);
        // await fetchAndDisplay(url);
    } catch (err) {
        console.error(err.message);
    }
}

function copyUserPromptToModelResponse() {
    modelResponse.innerHTML = userPrompt.value;
}
function getDayPart(currentTime) {
    // Get the hour (0-23)
    const hour = currentTime.getHours();

    // Define ranges for each day part
    const morning = { start: 5, end: 11 };
    const afternoon = { start: 12, end: 17 };
    const evening = { start: 18, end: 21 };
    const night = { start: 22, end: 4 };

    // Check which range the hour falls into
    if (hour >= morning.start && hour < afternoon.start) {
        return "Morning";
    } else if (hour >= afternoon.start && hour < evening.start) {
        return "Afternoon";
    } else if (hour >= evening.start && hour < night.start) {
        return "Evening";
    } else {
        return "Night";
    }
}

async function fetchAndDisplay(url) {
    const resp = await fetch(url);
    if (!resp.ok) { throw new Error(`response status: ${resp.status}`) }
    const respTxt = await resp.text();
    sessionStorage.setItem("ragDocs", respTxt);
    embeddingResponse.innerHTML = marked.parse(respTxt);
}

function showStorage() {
    console.log(sessionStorage.getItem("latlng"));
    console.log(sessionStorage.getItem("neighborhood"));
}
showStorage();

const meaningOfLifeLink = document.getElementById("meaningOfLife");
meaningOfLifeLink.addEventListener("click", molStreamer);
async function molStreamer() {
    modelResponse.innerText = "";
    embeddingResponse.innerHTML = "";
    const url = `/life?loc=${encodeURIComponent(sessionStorage.getItem("neighborhood"))}&latlng=${sessionStorage.getItem("latlng")}`;
    await streamToElement(modelResponse, url);
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

const selectedContext = document.getElementById("selected-context");
selectedContext.addEventListener("change", (ev) => { recordSelectedContext(ev.target.value) })

function recordSelectedContext(ctx) {
    sessionStorage.setItem("context", ctx);
    console.log(`set context to ${ctx}`);
}
recordSelectedContext("General");

const personalLogBtn = document.getElementById("personalLog");
const activitiesBtn = document.getElementById("activities");
const activitiesDiv = document.getElementById("activitiesScreen");
activitiesDiv.classList.add("hide");

personalLogBtn.addEventListener("click", () => {
    window.location.replace("/personallog");
});

activitiesBtn.addEventListener("click", showActivitiesScreen);

const selectedSuggestion = document.getElementById("selected-suggestion");
selectedSuggestion.addEventListener("change", (ev) => {
    copySelectedSuggestionToUserPrompt(ev.target.value)
});
function copySelectedSuggestionToUserPrompt(prompt) {
    userPrompt.value = prompt;
    console.log(prompt);
}

function showActivitiesScreen() {
    activitiesDiv.classList.remove("hide");
} 