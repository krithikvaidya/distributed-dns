#!/bin/bash

sudo apt update

cd /home/ubuntu
mkdir -p workspace
cd workspace

printf "\nPerforming first time setup.\n"

echo -n "Enter replica id: "
read userinput

echo -n "Enter the gRPC server addresses of all replicas: "
read addr0
read addr1
read addr2

# install unzip utility
sudo apt-get install unzip

# install go
wget -q -O - https://git.io/vQhTU | bash

# download replicated key value store files
curl -L "https://docs.google.com/uc?export=download&id=1AoU6RvydJqmg0j7M1OP7CNYkVoPaEAnC" --output program_files.zip
unzip program_files.zip -d replicated_kv_store
rm program_files.zip

echo "" >> ~/.bashrc

echo "export REPLICA_ID=${userinput}" >> ~/.bashrc
echo "export REP_0_GRPC_ADDR=${addr0}" >> ~/.bashrc
echo "export REP_1_GRPC_ADDR=${addr1}" >> ~/.bashrc
echo "export REP_2_GRPC_ADDR=${addr2}" >> ~/.bashrc

echo "" >> ~/.bashrc

echo "cd ~/workspace/replicated_kv_store" >> ~/.bashrc
echo "go run . -n 3" ?? ~/.bashrc
