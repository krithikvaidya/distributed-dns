#!/bin/bash

sudo apt update

cd /home/ubuntu
mkdir -p workspace
cd workspace

printf "\nPerforming first time setup.\n"

# install unzip utility
sudo apt-get install unzip

# install aws cli utility
curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
unzip awscliv2.zip
sudo ./aws/install

# install go. reference: https://github.com/canha/golang-tools-install-script
wget -q -O - https://git.io/vQhTU | bash

# set environment variables for this shell instance
export GOROOT=/home/ubuntu/.go
export PATH=$GOROOT/bin:$PATH
export GOPATH=/home/ubuntu/go
export PATH=$GOPATH/bin:$PATH

# download replicated key value store files
git clone https://github.com/krithikvaidya/distributed-dns.git -b aws-dns --single-branch
cd distributed-dns/

go build