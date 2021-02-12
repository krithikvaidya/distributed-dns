#!/bin/bash

sudo apt update

cd /home/ubuntu
mkdir -p workspace
cd workspace

printf "\nPerforming first time setup.\n"

# install unzip utility
sudo apt-get install unzip

# install go
wget -q -O - https://git.io/vQhTU | bash

# download replicated key value store files

git clone https://github.com/krithikvaidya/distributed-dns.git -b aws --single-branch
cd distributed-dns/replicated_kv_store

export GOROOT=/home/ubuntu/.go
export PATH=$GOROOT/bin:$PATH
export GOPATH=/home/ubuntu/go
export PATH=$GOPATH/bin:$PATH

cd ~/workspace/replicated_kv_store
go build
