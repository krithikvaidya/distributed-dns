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
export REP_0_GRPC_ADDR=:5555
export REP_1_GRPC_ADDR=:5556
export REP_2_GRPC_ADDR=:5557

cd /home/ubuntu/workspace/replicated_kv_store
./replicated_kv_store -n 3

--//