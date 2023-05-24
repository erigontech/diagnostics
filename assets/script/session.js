
async function fetchContent(sessionName, description, url, divId) {
    const d = document.getElementById(divId);
    d.innerHTML = "Fetching " + description + " ...";
    var formData = new FormData();
    formData.append('current_sessionname', sessionName);
    const request =  createRequest(url, "POST", formData) 
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
    formData = createForm({
        'current_sessionname': sessionName,
        'file': filename,
        'size': size,
    })
    const request =  createRequest(url, "POST", formData) 
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
    const request =  createRequest("/ui/reorgs", "POST", formData) 
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
    const request =  createRequest("/ui/bodies_download", "POST", formData) 
    fetch(request)
        .then((response) => response.body)
        .then((body) => body.getReader())
        .then((reader) => processReaderReplace(d, reader))
        .then(() => console.log('completed'))
        .catch((err) => d.innerHTML = "ERROR: " + err.message);
}

async function headersDownload(sessionName) {
    const d = document.getElementById('headers_download');
    d.innerHTML = "Tracking headers download...";
    var formData = new FormData();
    formData.append('current_sessionname', sessionName);
    const request =  createRequest("/ui/headers_download", "POST", formData) 
    fetch(request)
        .then((response) => response.body)
        .then((body) => body.getReader())
        .then((reader) => processReaderReplace(d, reader))
        .then(() => console.log('completed'))
        .catch((err) => d.innerHTML = "ERROR: " + err.message);
}

async function fetchSession(sessionName, description, sessionPin) {
    var formData = null
    if(description == 'switch'){
        var current_session_name = document.getElementById("current_sessionname").value
        formData = createForm({
            "current_sessionname": current_session_name,
            "sessionname": "",
            "pin": "",
            [sessionPin]: sessionName
        })
    }
    else if(description == 'resume'){
        var newSessionName = document.getElementById("sessionname").value
        var newPin = document.getElementById("pin").value
        formData = createForm({
            "current_sessionname": sessionName,
            "sessionname": newSessionName,
            "resume_session": "Resume or take over operator session",
            "pin": newPin
        })
    }
    const request =  createRequest('/ui/', "POST", formData) 
    fetch(request)
    .then(response => response.text())
    .then(html => {
        document.body.innerHTML = html;
    })
}

function createForm(jsonData) {
    var formData = new FormData();
    for (const [key, value] of Object.entries(jsonData)) {
        formData.append(key, value)
    }
    return formData;
}

function createRequest(url, method, formData) {
    const request = new Request(url, {
        method: method,
        mode: "cors",
        cache: "default",
        body: new URLSearchParams(formData),
    });
    return  request
}