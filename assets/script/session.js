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

async function fetchLogHead(sessionName, filename, size) {
    console.log("Session = " + sessionName)
    console.log("Filename = " + filename)
    console.log("Size = " + size)
    const d = document.getElementById('log_part');
    d.innerHTML = "Fetch log head...";
    let formData = new FormData();
    formData.append('current_sessionname', sessionName);
    formData.append('filename', filename);
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

async function fetchLogTail(sessionName, filename, size) {
    console.log("Session = " + sessionName)
    console.log("Filename = " + filename)
    console.log("Size = " + size)
    const d = document.getElementById('log_part');
    d.innerHTML = "Fetch log head...";
    let formData = new FormData();
    formData.append('current_sessionname', sessionName);
    formData.append('filename', filename);
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
