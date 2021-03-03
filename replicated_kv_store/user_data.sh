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

export REPLICA_ID=
export REP_0_GRPC_ADDR=  # fill in, eg: ec2-23-21-123-241.compute-1.amazonaws.com:5000
export REP_1_GRPC_ADDR=
export REP_2_GRPC_ADDR=

export LB_ARN=  # ARN of respective network load balancer
export INST_ID_0=  # AWS ID of instance 0
export INST_ID_1=
export INST_ID_2=

cd /home/ubuntu/workspace/distributed-dns/replicated_kv_store
./replicated_kv_store -n 3

--//