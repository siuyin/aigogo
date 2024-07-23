const recLogBtn = document.getElementById("recordLog");
recLogBtn.addEventListener("click", () => { console.log("record start"); });

const recStopBtn = document.getElementById("recordStop");
recStopBtn.addEventListener("click", () => { console.log("record stop"); });

const reviewLatestLogLink = document.getElementById("reviewLatestLog");
reviewLatestLogLink.addEventListener("click", () => {
    console.log("reviewing log");
    logText.innerText = "The quick brown fox jumps over the lazy dog.";
})

const logText = document.getElementById("logText");

const saveEditedText = document.getElementById("saveEditedText");
saveEditedText.addEventListener("click", () => {
    summary.innerHTML = `summary: ${logText.value} <p>tags: Primary=${primaryHighlight.value}, Secondary=${secondaryHighlight.value}, People=${selectedPeople(whoIWasWith.selectedOptions)}`;
});

const summary = document.getElementById("summary");
const primaryHighlight = document.getElementById("primaryHighlight");
const secondaryHighlight = document.getElementById("secondaryHighlight");
const whoIWasWith = document.getElementById("whoIWasWith");

function selectedPeople(peoplelist) {
    let ret = [];
    for (const p of peoplelist) {
        ret.push(p.innerText);
    }
    return ret.join("|");
}