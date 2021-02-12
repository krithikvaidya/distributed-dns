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

export REPLICA_ID=  # fill in
export REP_0_GRPC_ADDR=  # fill in, eg: ec2-23-21-123-241.compute-1.amazonaws.com:5000
export REP_1_GRPC_ADDR=  # fill in
export REP_2_GRPC_ADDR=  # fill in

cd /home/ubuntu/workspace/distributed-dns/replicated_kv_store
./replicated_kv_store -n 3

--//