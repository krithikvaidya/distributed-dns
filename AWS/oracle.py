import os
import json

import boto3

from flask import Flask
from flask import request

app = Flask(__name__)

init_done = False
oracle = None

class Oracle:

    def create_load_balancers(self):

        self.TG_ARNs = {}

        for region in self.region_names:

            self.TG_ARNs[region] = []

            for i in range(self.instances_per_region//self.replicas_per_ns):
                
                lb = self.aws_clients[region].create_load_balancer(
                    Name=region + '-lb' + str(i),
                    Type='network',
                    Subnets=[
                        self.subnets[region]
                    ]
                )

                print (lb)

                tg = self.aws_clients[region].create_target_group(
                    Name=region + '-tg' + str(i),
                    Protocol='TCP',
                    Port=4000,
                    TargetType='instance',
                    VpcId=self.VPCs[region],
                )

                print(tg)

                self.TG_ARNs[region].append(tg['TargetGroups'][0]['TargetGroupArn'])

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

    def register_replica(self, data):

        inst_id = data['inst_id']
        region = data['region']
        internal_ip = data['internal_ip']

        for instance in self.instance_details[region]:
            if instance['inst_id'] == inst_id:
                return {"message": "Already registered this instance for this region."}

        if len(self.instance_details[region]) == self.instances_per_region:
            return {"message": "Cannot register more instances for this region."}

        self.instance_details[region].append({'inst_id': inst_id, 'internal_ip': internal_ip})
        return {"message": "Registered " + inst_id}


    def get_env(self, data):

        if len(self.instance_details[data['region']]) != self.instances_per_region:
            return {"message": "Waiting for more instances to register"}

        i = 0
        for instance in self.instance_details[data['region']]:
            if instance['inst_id'] == data['inst_id']:
                break

            i += 1

        replica_id =  (i % self.instances_per_region)

        internal_ips = []
        instance_ids = []

        for j in range(self.instances_per_region):
            internal_ips.append(self.instance_details[data['region']][(self.instances_per_region * (i / self.instances_per_region)) + j]['internal_ip'])
            instance_ids.append(self.instance_details[data['region']][(self.instances_per_region * (i / self.instances_per_region)) + j]['inst_id'])

        return {
            'n_replicas': self.replicas_per_ns,
            'replica_id': replica_id,
            'internal_ips': internal_ips,
            'tg_arn': self.TG_ARNs[data['region']][i/self.instances_per_region],
            'instance_ids': instance_ids
        }
                
    
    def __init__(self, data):

        self.n_regions = int(data["n_regions"])
        self.region_names = data["region_names"]
        self.instances_per_region = int(data["instances_per_region"])
        self.replicas_per_ns = int(data["replicas_per_ns"])
        self.aws_clients = {}
        self.instance_details = {}
        self.VPCs = {}
        self.subnets = {}

        for region in self.region_names:
            self.instance_details[region] = []
            self.aws_clients[region] = boto3.client('elbv2', region_name=region)

            # For getting default subnet and vpc
            ec2_client = boto3.client('ec2', region_name=region)

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

        self.create_load_balancers()


@app.route('/init', methods=['POST'])
def init():

    global init_done
    global oracle

    print("Got initialization request. Request data:")
    data = request.get_json()
    print(data)

    if init_done:
        print("Initialization already done!")
        return {"message": "Initialization already done!"}

    oracle = Oracle(data)

    init_done = True
    return {"message": "Initialization Successful!"}


@app.route('/register_replica', methods=['POST'])
def register_replica():

    global oracle
    data = request.get_json()

    return oracle.register_replica(data)


@app.route('/get_env', methods=['POST'])
def get_env():

    global oracle
    data = request.get_json()

    return oracle.get_env(data)


@app.route('/clear_all', methods=['GET'])
def clear_all():
    global init_done
    global oracle

    del oracle
    init_done = False

    return {"message": "Successfully cleared data"}


if __name__ == '__main__':
    app.run()