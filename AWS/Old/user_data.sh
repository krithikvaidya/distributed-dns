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

export N_REPLICA=
export REPLICA_ID= # 0, 1, 2...

export REP_0_GRPC_ADDR= # internal IP of replica 0
export REP_1_GRPC_ADDR=
export REP_2_GRPC_ADDR=

# store ARN of respective target group. eg: arn:aws:elasticloadbalancing:us-east-1:991577021500:targetgroup/use1-root-tg/690a4ae0ae3d5f99
export TG_ARN=

# store AWS instance IDs.
export INST_ID_0= # eg: i-016efc93518986add
export INST_ID_1=
export INST_ID_2=

cd /home/ubuntu/workspace/distributed-dns/replicated_kv_store
./replicated_kv_store

--//