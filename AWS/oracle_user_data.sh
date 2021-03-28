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

    runuser -l ubuntu -c 'curl -L "https://gist.githubusercontent.com/krithikvaidya/208f50c2ef39ff7cdea4f388cbaf0883/raw" --output oracle.py'
    runuser -l ubuntu -c 'sudo apt update'
    runuser -l ubuntu -c 'sudo apt install python3-pip -y'
    runuser -l ubuntu -c 'pip3 install boto3 flask'

    touch init_done
fi

runuser -l ubuntu -c 'python3 oracle.py'