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

runuser -l ubuntu -c 'curl -L "https://gist.githubusercontent.com/krithikvaidya/a99c6beb27a3f85e8161073216f068c8/raw" --output ec2_onetime_setup.sh'
runuser -l ubuntu -c 'chmod +x ec2_onetime_setup.sh'
runuser -l ubuntu -c './ec2_onetime_setup.sh'

runuser -l ubuntu -c 'cd /home/ubuntu/workspace/distributed-dns/replicated_kv_store && ./replicated_kv_store -n 3'