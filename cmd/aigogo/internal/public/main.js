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
        geoCoords.innerText = `I see you are in ${respTxt}`;
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

const modelResponse = document.getElementById("modelResponse");
function copyUserPromptToModelResponse() {
    modelResponse.innerHTML = userPrompt.value;
}

async function retrieveDocsForAugmentation() {
    let usrQry = encodeURIComponent(userPrompt.value);
    //     const url = `http://localhost:8080/retr?userPrompt=${usrQry}`; // DEV
    const url = `https://aigogo-onsvm4sjba-uc.a.run.app/userPrompt?latlng=${usrQry}` // PROD
    try {
        const resp = await fetch(url);
        if (!resp.ok) { throw new Error(`response status: ${resp.status}`) }
        const respTxt = await resp.text();
        sessionStorage.setItem("ragDocs", respTxt);
        modelResponse.innerHTML = respTxt;
    } catch (err) {
        console.error(err.message);
    }

}

function showStorage() {
    console.log(sessionStorage.getItem("latlng"));
    console.log(sessionStorage.getItem("neighborhood"));
}
showStorage();