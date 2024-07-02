const geoCoords = document.getElementById("geoCoords");
if (!("geolocation" in navigator)) {
    console.log("geo location API not available");
}

navigator.geolocation.getCurrentPosition((position) => {
    geoCoords.innerText = `I see you are roughly located at coordinates: (${position.coords.latitude}, ${position.coords.longitude})`;
    updateNeighborhood(position.coords.latitude, position.coords.longitude);
    sessionStorage.setItem("latlng", `${position.coords.latitude},${position.coords.longitude}`);
});

async function updateNeighborhood(lat, lng) {
    //     const url = `http://localhost:8080/loc?latlng=${lat},${lng}`; // DEV
    const url = `https://aigogo-onsvm4sjba-uc.a.run.app/loc?latlng=${lat},${lng}` // PROD
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

const currentTime = document.getElementById("currentTime");
currentTime.innerText = `Good ${getDayPart(new Date)}`;

const userPrompt = document.getElementById("userPrompt");
const userSubmit = document.getElementById("userSubmit");
userSubmit.addEventListener("click", retrieveDocsForAugmentation);

const embeddingResponse = document.getElementById("embeddingResponse");
const modelResponse = document.getElementById("modelResponse");
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
async function retrieveDocsForAugmentation() {
    const loc = encodeURIComponent(sessionStorage.getItem("neighborhood"));
    let usrQry = encodeURIComponent(userPrompt.value);
    //     const url = `http://localhost:8080/retr?userPrompt=${usrQry}&loc=${loc}&latlng=${sessionStorage.getItem("latlng")}`; // DEV
    const url = `https://aigogo-onsvm4sjba-uc.a.run.app/retr?userPrompt=${usrQry}&loc=${loc}&latlng=${sessionStorage.getItem("latlng")}` // PROD

    embeddingResponse.innerHTML = "working ... give me a few seconds ..."
    try {
        const resp = await fetch(url);
        if (!resp.ok) { throw new Error(`response status: ${resp.status}`) }
        const respTxt = await resp.text();
        sessionStorage.setItem("ragDocs", respTxt);
        embeddingResponse.innerHTML = respTxt;
        modelResponse.innerText = "";
    } catch (err) {
        console.error(err.message);
    }

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
    const url = `/life?loc=${encodeURIComponent(sessionStorage.getItem("neighborhood"))}`;
    const res = await fetch(url);
    const dec = new TextDecoder("utf-8");
    for await (const chunk of res.body) {
        modelResponse.innerHTML += (dec.decode(chunk));
    }
}