GOOS=linux GOARC=amd64 go build -o staging-server
ssh vektorprogrammet@82.196.15.63 'sudo service staging-server stop'
scp staging-server vektorprogrammet@82.196.15.63:/var/www/staging-server
ssh vektorprogrammet@82.196.15.63 'sudo service staging-server start'
