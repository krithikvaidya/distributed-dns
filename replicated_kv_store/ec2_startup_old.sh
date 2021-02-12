#!/bin/bash

sudo apt update

cd /home/ubuntu
mkdir -p workspace
cd workspace

# check if its first start up
FILE=init_done
if ! test -f "$FILE"; then

    printf "\nPerforming first time setup.\n"

    # install unzip utility
    sudo apt-get install unzip

    # install go
    wget -q -O - https://git.io/vQhTU | bash
    # source /home/ubuntu/.bashrc
    # source /.bashrc

    curl -L "https://docs.google.com/uc?export=download&id=1AoU6RvydJqmg0j7M1OP7CNYkVoPaEAnC" --output program_files.zip
    unzip program_files.zip -d replicated_kv_store
    rm program_files.zip

    # create init_done file
    touch init_done

fi

export GOROOT=/.go
export PATH=$GOROOT/bin:$PATH
export GOPATH=/go
export PATH=$GOPATH/bin:$PATH
export REPLICA_ID=0

echo export GOROOT=/.go >> /etc/profile
echo export PATH=$GOROOT/bin:$PATH >> /etc/profile
echo export GOPATH=/go >> /etc/profile
echo export PATH=$GOPATH/bin:$PATH >> /etc/profile
echo export REPLICA_ID=0 >> /etc/profile

cd replicated_kv_store

go run . -n 3

