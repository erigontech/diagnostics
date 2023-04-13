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
