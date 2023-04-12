async function fetchVersions(sessionName) {
    const d = document.getElementById('versions');
    d.innerHTML = "Fetching command line args...";
    var formData = new FormData();
    formData.append('current_sessionname', sessionName);
    const request = new Request("/ui/versions", {
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

async function fetchCmdLineArgs(sessionName) {
    const d = document.getElementById('cmdlineargs');
    d.innerHTML = "Fetching command line args...";
    var formData = new FormData();
    formData.append('current_sessionname', sessionName);
    const request = new Request("/ui/cmd_line", {
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

async function fetchLogList(sessionName) {
    const d = document.getElementById('loglist');
    d.innerHTML = "Fetching log list...";
    let formData = new FormData();
    formData.append('current_sessionname', sessionName);
    const request = new Request("/ui/log_list", {
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

async function fetchLogHead(logPartId, sessionName, filename, size) {
    const d = document.getElementById(logPartId);
    d.innerHTML = "Fetch log head...";
    let formData = new FormData();
    formData.append('current_sessionname', sessionName);
    formData.append('file', filename);
    formData.append('size', size);
    const request = new Request("/ui/log_head", {
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

async function fetchLogTail(logPartId, sessionName, filename, size) {
    const d = document.getElementById(logPartId);
    d.innerHTML = "Fetch log head...";
    let formData = new FormData();
    formData.append('current_sessionname', sessionName);
    formData.append('file', filename);
    formData.append('size', size);
    const request = new Request("/ui/log_tail", {
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