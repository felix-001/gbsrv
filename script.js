function run() {
    startServer("")
}

// This is a JavaScript function that we will call from Go
// in main.go.
function setDivContent(divId, content) {
    document.getElementById(divId).innerHTML = content
}
