runuser -l ubuntu -c 'curl -L "https://gist.githubusercontent.com/krithikvaidya/208f50c2ef39ff7cdea4f388cbaf0883/raw" --output oracle.py'
runuser -l ubuntu -c 'sudo apt update'
runuser -l ubuntu -c 'sudo apt install python3-pip -y'
runuser -l ubuntu -c 'pip3 install boto3 flask'
runuser -l ubuntu -c 'python3 oracle.py'

