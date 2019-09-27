drop();

function drop() {
    let headers = {
        "file": "sandcat.go",
        "platform": determinePlatform()
    };
    let loc = location.protocol+'//'+location.hostname+(location.port ? ':'+location.port: '');
    fetch(loc + '/file/download', {
        method: "POST",
        headers: headers})
      .then(resp => resp.blob())
      .then(blob => {
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.style.display = 'none';
        a.href = url;
        a.download = 'sandcat';
        document.body.appendChild(a);
        a.click();
        window.URL.revokeObjectURL(url);
      }).catch(() => console.log("Error auto-downloading 54ndc47"));
}


function determinePlatform() {
    let OSName = "unknown";
    if (window.navigator.userAgent.startsWith("Windows")!= -1) OSName="windows";
    if (window.navigator.userAgent.indexOf("Mac") != -1) OSName="darwin";
    if (window.navigator.userAgent.indexOf("X11") != -1) OSName="linux";
    if (window.navigator.userAgent.indexOf("Linux") != -1) OSName="linux";
    return OSName;
}
