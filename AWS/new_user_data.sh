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

FILE=init_done
if test -f "$FILE"; then
    echo "$FILE exists. Skipping initialization"
else
    echo "$FILE does not exist. Performing initialization"

    runuser -l ubuntu -c 'curl -L "https://gist.githubusercontent.com/krithikvaidya/a99c6beb27a3f85e8161073216f068c8/raw" --output ec2_onetime_setup.sh'
    runuser -l ubuntu -c 'chmod +x ec2_onetime_setup.sh'
    runuser -l ubuntu -c './ec2_onetime_setup.sh'

    touch init_done

fi

runuser -l ubuntu -c 'export ORACLE_ADDR=ec2-44-192-119-167.compute-1.amazonaws.com && cd /home/ubuntu/workspace/distributed-dns/ && ./distributed-dns'
