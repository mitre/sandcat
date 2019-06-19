# 54ndc47

This plugin contains:
* A custom agent, with variants for Windows, Linux and MacOSX
* API endpoints for the agent to communicate to CALDERA over HTTPS

## Quick start

Start the agent on any operating system

**OSX**:
```
while true; do curl -sk -X POST -H 'file:sandcat-darwin' http://localhost:8888/file/download > /tmp/sandcat-darwin && chmod +x /tmp/sandcat-darwin && /tmp/sandcat-darwin http://localhost:8888 my_group; sleep 60; done
```

**Linux**:
```
while true; do curl -sk -X POST -H 'file:sandcat-linux' http://localhost:8888/file/download > /tmp/sandcat-linux && chmod +x /tmp/sandcat-linux && /tmp/sandcat-linux http://localhost:8888 my_group; sleep 60; done
```

**Windows**:
```
while($true) {$url="http://localhost:8888/file/download";$wc=New-Object System.Net.WebClient;$wc.Headers.add("file","sandcat.exe");$output="C:\Users\Public\sandcat.exe";$wc.DownloadFile($url,$output);C:\Users\Public\sandcat.exe http://localhost:8888 my_group; sleep 60}
```

## Updates

Make a change to the sandcat.go code? Run one of the following to recompile, then put the resulting sandcat file in the
stockpile plugin payloads directory
```
GOOS=windows go build -ldflags="-s -w" sandcat.go
GOOS=linux go build -ldflags="-s -w" sandcat.go
GOOS=darwin go build -ldflags="-s -w" sandcat.go
```