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
currentTime.innerText = `The time is: ${new Date(Date.now()).toString()}`;

const userPrompt = document.getElementById("userPrompt");
const userSubmit = document.getElementById("userSubmit");
userSubmit.addEventListener("click", retrieveDocsForAugmentation);

const embeddingResponse = document.getElementById("embeddingResponse");
const modelResponse = document.getElementById("modelResponse");
function copyUserPromptToModelResponse() {
    modelResponse.innerHTML = userPrompt.value;
}

async function retrieveDocsForAugmentation() {
    const loc = encodeURIComponent(sessionStorage.getItem("neighborhood"));
    let usrQry = encodeURIComponent(userPrompt.value);
    //     const url = `http://localhost:8080/retr?userPrompt=${usrQry}&loc=${loc}`; // DEV
    const url = `https://aigogo-onsvm4sjba-uc.a.run.app/retr?userPrompt=${usrQry}&loc=${loc}` // PROD

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