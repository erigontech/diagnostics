async function fetchContent(sessionName, description, url, divId) {
    const d = document.getElementById(divId);
    d.innerHTML = "Fetching " + description + " ...";
    var formData = new FormData();
    formData.append('current_sessionname', sessionName);
    const request = new Request(url, {
        method: "POST",
        mode: "cors",
        cache: "default",
        body: new URLSearchParams(formData),
    });
    try {
        const response = await fetch(request);
        if (!response.ok) {
            d.innerHTML = "ERROR: Network response was not OK";
            return
        }
        const result = await response.text();
        d.innerHTML = result
    } catch (error) {
        d.innerHTML = "ERROR: " + error.message
    }
}

async function fetchLogPart(logPartId, sessionName, description, url, filename, size) {
    const d = document.getElementById(logPartId);
    d.innerHTML = "Fetch " +description + " ...";
    let formData = new FormData();
    formData.append('current_sessionname', sessionName);
    formData.append('file', filename);
    formData.append('size', size);
    const request = new Request(url, {
        method: "POST",
        mode: "cors",
        cache: "default",
        body: new URLSearchParams(formData),
    });
    try {
        const response = await fetch(request);
        if (!response.ok) {
            d.innerHTML = "ERROR: Network response was not OK";
            return
        }
        const result = await response.text();
        d.innerHTML = result
    } catch (error) {
        d.innerHTML = "ERROR: " + error.message
    }
}

async function clearLog(logPartId) {
    const d = document.getElementById(logPartId);
    d.innerHTML = ""
}

async function processReader(d, reader) {
    var first = true
    const utf8Decoder = new TextDecoder("utf-8");
    var buffer = ''
    for (let chunk = await reader.read();!chunk.done;chunk = await reader.read()) {
        if (!chunk.done) {
            buffer += utf8Decoder.decode(chunk.value, { stream: false });
        }
        var lastLineBreak = buffer.lastIndexOf('\n')
        if (lastLineBreak != -1) {
            if (first) {
                d.innerHTML = buffer.substring(0, lastLineBreak);
                first = false
            } else {
                d.innerHTML += buffer.substring(0, lastLineBreak);
            }
            buffer = buffer.substring(lastLineBreak + 1)
        }
    }
    if (first) {
        d.innerHTML = buffer;
        first = false;
    } else {
        d.innerHTML += buffer;
    }
}

async function findReorgs(sessionName) {
    const d = document.getElementById('reorgs');
    d.innerHTML = "Looking for reorgs...";
    var formData = new FormData();
    formData.append('current_sessionname', sessionName);
    const request = new Request("/ui/reorgs", {
        method: "POST",
        mode: "cors",
        cache: "default",
        body: new URLSearchParams(formData),
    });
    fetch(request)
        .then((response) => response.body)
        .then((body) => body.getReader())
        .then((reader) => processReader(d, reader))
        .then(() => console.log('completed'))
        .catch((err) => d.innerHTML = "ERROR: " + err.message);
}

async function bodiesDownload(sessionName) {
    const d = document.getElementById('bodies_download');
    d.innerHTML = "Tracking bodies download...";
    var formData = new FormData();
    formData.append('current_sessionname', sessionName);
    const request = new Request("/ui/bodies_download", {
        method: "POST",
        mode: "cors",
        cache: "default",
        body: new URLSearchParams(formData),
    });
    fetch(request)
        .then((response) => response.body)
        .then((body) => body.getReader())
        .then((reader) => processReader(d, reader))
        .then(() => console.log('completed'))
        .catch((err) => d.innerHTML = "ERROR: " + err.message);
}