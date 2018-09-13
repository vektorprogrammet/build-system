#!/usr/bin/env bash
GOOS=linux GOARC=amd64 go build -o staging-server
if [ $? -eq 0 ]; then
    ssh vektorprogrammet@82.196.15.63 'sudo service staging-server stop'
    scp staging-server vektorprogrammet@82.196.15.63:/var/www/staging-server
    ssh vektorprogrammet@82.196.15.63 'sudo service staging-server start'
else
    echo "Build was unsuccessful"
fi
