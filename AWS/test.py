
import boto3

client = boto3.client('globalaccelerator',region_name='us-west-2')

response = client.create_accelerator(
    Name='ga-5',
    IpAddressType='IPV4',
    Enabled=True,
)

print(response['Accelerator']['AcceleratorArn'])

response = client.create_listener(
    AcceleratorArn=response['Accelerator']['AcceleratorArn'],
    PortRanges=[
        {
            'FromPort': 4000,
            'ToPort': 4000
        },
    ],
    Protocol='TCP',
)

print(response)

for j in range(1):
    response = client.create_endpoint_group(
        ListenerArn=response['Listener']['ListenerArn'],
        EndpointGroupRegion='us-east-1',
        EndpointConfigurations=[
            {
                'EndpointId': 'arn:aws:elasticloadbalancing:us-east-1:991577021500:loadbalancer/net/testtlb/573adf19944bc139',
            },
        ],
    )
    
    print(response)