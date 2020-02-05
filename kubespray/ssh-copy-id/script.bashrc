#!/bin/bash
set -x # enable bash debug mode
if [ -s vassal-public-key.pub ]; then # if file exists and not empty
    for ip in `cat server.txt`; do # for each line from the file
        # add EOL to the end of the file and echo it into ssh, where it is added to the authorized_keys
        sed -e '$s/$/\n/' -s vassal-public-key.pub | ssh $ip 'cat >> ~/.ssh/authorized_keys'
    done
else
    echo "Put new vassal public key into ./vassal-public-key.pub to add it to server.txt hosts"
fi
