Content-Type: multipart/mixed; boundary="//"
MIME-Version: 1.0

--//
Content-Type: text/cloud-config; charset="us-ascii"
MIME-Version: 1.0
Content-Transfer-Encoding: 7bit
Content-Disposition: attachment; filename="cloud-config.txt"

#cloud-config
cloud_final_modules:
- [scripts-user, always]

--//
Content-Type: text/x-shellscript; charset="us-ascii"
MIME-Version: 1.0
Content-Transfer-Encoding: 7bit
Content-Disposition: attachment; filename="userdata.txt"

#!/bin/bash

export REPLICA_ID=0
export REP_0_GRPC_ADDR=172.31.71.187  # fill in, eg: ec2-23-21-123-241.compute-1.amazonaws.com:5000
export REP_1_GRPC_ADDR=172.31.73.39
export REP_2_GRPC_ADDR=172.31.71.70

# store ARN of respective load balancer.
export LB_ARN=arn:aws:elasticloadbalancing:us-east-1:991577021500:loadbalancer/net/use1-root-lb/40476ec08960165f

# store instance IDs.
export INST_ID_0=i-04c1194f6be364f3c
export INST_ID_1=i-09e7b657dfda2c807
export INST_ID_2=i-00e369eb7c4a8ae9e

cd /home/ubuntu/workspace/distributed-dns/replicated_kv_store
./replicated_kv_store -n 3

--//