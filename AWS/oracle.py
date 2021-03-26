import os
import boto3

from flask import Flask
from flask import request

app = Flask(__name__)

init_done = False
oracle = None
class Oracle:

    def create_load_balancers(self):

        self.LBs = {}
        self.TGs = {}

        for region in self.region_names:

            self.LBs[region] = []
            self.TGs[region] = []

            for i in range(self.instances_per_region/self.replicas_per_ns):
                
                lb = self.aws_clients[region].create_load_balancer(
                    Name=region + '-lb' + str(i),
                    Type='network',
                    Subnets=[
                        self.subnets[region]
                    ]
                )

                print (lb)

                self.LBs[region].append(lb)

                tg = self.aws_clients[region].create_target_group(
                    Name=region + '-tg' + str(i),
                    Protocol='TCP',
                    Port=4000,
                    TargetType='instance',
                    VpcId=self.VPCs[region],
                )

                print(tg)

                self.TGs[region].append(tg)

                response = self.aws_clients[region].modify_target_group_attributes(
                    Attributes=[
                        {
                            'Key': 'deregistration_delay.timeout_seconds',
                            'Value': '1',
                        },
                    ],
                    TargetGroupArn=tg['TargetGroups'][0]['TargetGroupArn'],
                )

                print(response)

                response = self.aws_clients[region].create_listener(
                    LoadBalancerArn=lb['LoadBalancers'][0]['LoadBalancerArn'],
                    Protocol='TCP',
                    Port=4000,
                    DefaultActions=[
                        {
                            'Type': 'forward',
                            'TargetGroupArn': tg['TargetGroups'][0]['TargetGroupArn']
                        }
                    ]
                )

                print(response)

                # response = client.register_targets(
                #     TargetGroupArn=tg['TargetGroups'][0]['TargetGroupArn'],
                #     Targets=[
                #         {
                #             'Id': 'i-08869900d12c4219e',
                #             'Port': 4000
                #         },
                #     ]
                # )


                
    
    def __init__(self, data):
        
        self.n_regions = int(data["n_regions"])
        self.region_names = data["region_names"]
        self.instances_per_region = int(data["instances_per_region"])
        self.replicas_per_ns = int(data["replicas_per_ns"])
        self.aws_clients = {}
        self.instance_details = {}
        self.VPCs = {}
        self.subnets = {}

        # For getting default subnet and vpc
        ec2_client = boto3.client('ec2', region_name="us-east-1")

        for region in region_names:
            self.instance_details[region] = {}
            self.aws_clients[region] = boto3.client('elbv2', region_name=region)

            response = ec2_client.describe_subnets(
                Filters=[
                    {
                        'Name': 'availabilityZone',
                        'Values': [
                            region + 'a',
                        ]
                    },
                ]
            )
            
            self.VPCs[region] = response['Subnets'][0]['VpcId']
            self.subnets[region] = response['Subnets'][0]['SubnetId']


@app.route('/init')
def init():

    global init_done

    print("Got initialization request. Request data:")
    data = request.get_json()
    print(data)

    if init_done:
        print("Initialization already done!")
        return "Initialization already done!"

    oracle = Oracle(data)

    init_done = True
    return "Initialization Successful!"


@app.route('/get_env')
def get_env():