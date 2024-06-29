const geoCoords = document.getElementById("geoCoords");
if (!("geolocation" in navigator)) {
    console.log("geo location API not available");
}

navigator.geolocation.getCurrentPosition((position) => {
    geoCoords.innerText = `I see you are roughly located at coordinates: (${position.coords.latitude}, ${position.coords.longitude})`;
    updateNeighborhood(position.coords.latitude,position.coords.longitude);
});

async function updateNeighborhood(lat,lng) {
    // const url = `http://localhost:8080/loc?latlng=${lat},${lng}`; // development
    const url = `https://aigogo-onsvm4sjba-uc.a.run.app/loc?latlng=${lat},${lng}` // production
    try {
        const resp = await fetch(url);
        if (!resp.ok) { throw new Error(`response status: ${resp.status}`) }
        const respTxt = await resp.text();
        geoCoords.innerText = `I see you are in ${respTxt}`;
    } catch (err) {
        console.error(err.message);
    }

}

const currentTime = document.getElementById("currentTime");
currentTime.innerText = `The time is: ${new Date(Date.now()).toString()}`;