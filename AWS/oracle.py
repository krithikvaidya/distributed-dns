import boto3

from flask import Flask
from flask import request

from flask_classful import FlaskView, route

app = Flask(__name__)

def create_load_balancers(aws_clients, n_regions, instances_per_region, replicas_per_ns):

    lb = client.create_load_balancer(
        Name='testtlb',
        Type='network',
        Subnets=[
            "subnet-09d2ef76208cc7dd6"
        ]
    )

class Oracle(FlaskView):

    # init_done = False
    # n_regions = 0
    # region_names = []
    # instances_per_region = 0
    # replicas_per_ns = 0
    # instance_details = {}

    def initialize(self):

        print("Got initialiazation request. Request data:")
        data = request.get_json()
        print(data)

        if self.init_done:
            print("Initialization already done!")
            return "Initialization already done!"
        
        self.n_regions = int(data["n_regions"])
        self.region_names = data["region_names"]
        self.instances_per_region = int(data["instances_per_region"])
        self.replicas_per_ns = int(data["replicas_per_ns"])
        self.aws_clients = {}
        self.instance_details = {}

        for region in region_names:
            self.instance_details[region] = {}
            self.aws_clients[region] = boto3.client('elbv2', region_name=region)

        self.init_done = True

    @app.route('/register_replica')
    def hello():

        return "Hello World!"


    @app.route('/get_env')
    def get_env():



    if __name__ == '__main__':
        app.run()

print (lb)

tg = client.create_target_group(
    Name='testttg',
    Protocol='TCP',
    Port=4000,
    TargetType='instance',
    VpcId="vpc-0fcc2864a35276e96",
)

print(tg)

response = client.modify_target_group_attributes(
    Attributes=[
        {
            'Key': 'deregistration_delay.timeout_seconds',
            'Value': '1',
        },
    ],
    TargetGroupArn=tg['TargetGroups'][0]['TargetGroupArn'],
)

print(response)

response = client.create_listener(
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

response = client.register_targets(
    TargetGroupArn=tg['TargetGroups'][0]['TargetGroupArn'],
    Targets=[
        {
            'Id': 'i-08869900d12c4219e',
            'Port': 4000
        },
    ]
)