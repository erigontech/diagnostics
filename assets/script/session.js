async function fetchCmdLineArgs(sessionName) {
    var area = document.getElementById('cmdlineargs');
    area.value = "Fetching command line args...";
    const request = new Request("/ui/cmdline", {
        method: "POST",
        mode: "cors",
        cache: "default",
        body: sessionName + "\n",
    });
    try {
        const response = await fetch(request);
        if (!response.ok) {
            area.value = "ERROR: Network response was not OK";
            return
        }
        const result = await response.text();
        area.value = result
    } catch (error) {
        area.value = "ERROR: " + error.message
    }
}
