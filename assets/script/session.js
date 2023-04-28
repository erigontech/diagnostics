
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

async function processReaderReplace(d, reader) {
    const utf8Decoder = new TextDecoder("utf-8");
    var buffer = ''
    for (let chunk = await reader.read();!chunk.done;chunk = await reader.read()) {
        if (!chunk.done) {
            buffer += utf8Decoder.decode(chunk.value, { stream: false });
        }
        var lastLineBreak = buffer.lastIndexOf('\n')
        if (lastLineBreak != -1) {
            lines = buffer.substring(0, lastLineBreak).split('\n')
            buffer = buffer.substring(lastLineBreak + 1)
            d.innerHTML = buffer.substring(0, lastLineBreak);
            lines.forEach(line => {
                d.innerHTML = line;
            });
        }
    }
    d.innerHTML = buffer;
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
        .then((reader) => processReaderReplace(d, reader))
        .then(() => console.log('completed'))
        .catch((err) => d.innerHTML = "ERROR: " + err.message);
}

async function switchSession(sessionName, sessionPin) {
    var formData = new FormData();
    var current_session_name = document.getElementById("current_sessionname").value
    var url = '/ui/switch_session'
    formData.append("current_sessionname", current_session_name);
    formData.append("sessionname", "");
    formData.append("pin", "");
    formData.append(sessionPin, sessionName);
    console.log(formData)
    const request = new Request(url, {
        method: "POST",
        mode: "cors",
        cache: "default",
        body: new URLSearchParams(formData),
    });
    fetch(request)
    .then(response => response.text())
    .then(html => {
        // replace entire page with rendered HTML template
        document.body.innerHTML = html;
    })
}
async function newSession() {
    var formData = new FormData();
    var session_name = document.getElementById("sessionname").value
    var current_sessionname = document.getElementById("current_sessionname").value
    var url = '/ui/'
    formData.append("current_sessionname ", current_sessionname );
    formData.append("sessionname", session_name);
    formData.append("new_session", "New operator session");
    formData.append("pin", "");
    console.log(formData)
    const request = new Request(url, {
        method: "POST",
        mode: "cors",
        cache: "default",
        body: new URLSearchParams(formData),
    });
    fetch(request)
    .then(response => response.text())
    .then(html => {
        document.body.innerHTML = html;
    })
}

async function resumeSession(sessionName) {
    var newSessionName = document.getElementById("sessionname").value
    var newPin = document.getElementById("pin").value

    var formData = new FormData();
    formData.append("current_sessionname", sessionName);
    formData.append("sessionname", newSessionName);
    formData.append("resume_session", "Resume or take over operator session");
    formData.append("pin", newPin);
    const request = new Request('/ui/', {
        method: "POST",
        mode: "cors",
        cache: "default",
        body: new URLSearchParams(formData),
    });
    fetch(request)
    .then(response => response.text())
    .then(html => {
        document.body.innerHTML = html;
    })
}