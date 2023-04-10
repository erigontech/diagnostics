async function fetchCmdLineArgs(sessionName) {
    var p = document.getElementById('cmdlineargs');
    p.innerHTML = "Fetching command line args...";
    const request = new Request("/ui/cmd_line", {
        method: "POST",
        mode: "cors",
        cache: "default",
        body: sessionName + "\n",
    });
    try {
        const response = await fetch(request);
        if (!response.ok) {
            p.innerHTML = "ERROR: Network response was not OK";
            return
        }
        const result = await response.text();
        p.innerHTML = result
    } catch (error) {
        p.innerHTML = "ERROR: " + error.message
    }
}

async function fetchLogList(sessionName) {
    var p = document.getElementById('loglist');
    p.innerHTML = "Fetching log list...";
    const request = new Request("/ui/log_list", {
        method: "POST",
        mode: "cors",
        cache: "default",
        body: sessionName + "\n",
    });
    try {
        const response = await fetch(request);
        if (!response.ok) {
            p.innerHTML = "ERROR: Network response was not OK";
            return
        }
        const result = await response.text();
        p.innerHTML = result
    } catch (error) {
        p.innerHTML = "ERROR: " + error.message
    }
}

async function fetchLogHead(sessionName, filename, size) {
    console.log("Session = " + sessionName)
    console.log("Filename = " + filename)
    console.log("Size = " + size)
}
