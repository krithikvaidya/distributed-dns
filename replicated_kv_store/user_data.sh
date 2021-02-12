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

curl -L "https://docs.google.com/uc?export=download&id=1t3yrnTErOzdB5IVUxb91MJGAPlBxkciR" --output ec2_startup2.sh

chmod 777 ec2_startup2.sh

su ubuntu -s ./ec2_startup2.sh

--//