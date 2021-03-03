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
export REP_0_GRPC_ADDR=172.31.74.22  # fill in, eg: ec2-23-21-123-241.compute-1.amazonaws.com:5000
export REP_1_GRPC_ADDR=172.31.78.208
export REP_2_GRPC_ADDR=172.31.70.118

# store ARN of respective load balancer.
export LB_ARN=arn:aws:elasticloadbalancing:us-east-1:991577021500:loadbalancer/net/use1-root-lb/40476ec08960165f

# store instance IDs.
export INST_ID_0=i-03e4aedd040c6675e
export INST_ID_1=i-0b89e17038e91ed04
export INST_ID_2=i-026c8bb28b265c37d

cd /home/ubuntu/workspace/distributed-dns/replicated_kv_store
./replicated_kv_store -n 3

--//