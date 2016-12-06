CloudFormation customizations
=============================

Porter allows services to define the subset of a CloudFormation (CFN) template
that they want to be different than what porter creates by default.

The basic rule is that porter won't alter defined resources and properties on
those resources. There are 2 major exception to this rule:

1. For properties that are lists like EC2 tags and SecurityGroups porter only
   appends to these lists
1. All the AutoScalingGroup's SecurityGroup's SecurityGroupEgress properties are
   overwritten with the value of [`security_group_egress`](config-reference.md#security_group_egress)

Some simple examples are below. Porter injects various [CFN parameters](http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/parameters-section-structure.html)
such as `PorterServiceName` and `PorterEnvironment`. Additionally there are a
number of [intrinsic functions](http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/intrinsic-function-reference.html)
and [psuedo parameters](http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/pseudo-parameter-reference.html)
that can provide additional data and make all sort of template customizations
possible, including creating [conditional resources](http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/conditions-section-structure.html)

Some projects are finding they can use a single template to deploy all of their
microservcies, which don't vary much from one another.

Finally, with porter hooks, you could even create a CFN template on the fly as
part of a deployment pipeline. See the [pre-pack hook](hooks/pre-pack.md) for
more.

Examples
--------

For all examples we'll use the following `.porter/config` and just change
`.porter/cfn.json` in each example.

```yaml
environments:
- name: dev

  stack_definition_path: .porter/cfn.json
```

Examples
- [SSH](#ssh)
- [Custom AMIs](#custom-amis)
- [Additional EC2 permissions](#additional-ec2-permissions)

### SSH

This is probably the first customization you'll want as you begin to explore a
porter deployment: SSH access. Porter adds all the resources _not_ already
defined in order to create a working CloudFormation template which means the
`AWS::AutoScaling::LaunchConfiguration` isn't overwritten, and security groups
are added to the `SecurityGroups` property already defined.

_Note: this is the complete CloudFormation template_

`.porter/cfn.json`

```json
{
  "Resources": {
    "AutoScalingLaunchConfiguration": {
      "Type": "AWS::AutoScaling::LaunchConfiguration",
      "Properties": {
        "SecurityGroups": [
          {
            "Ref": "SSHSecurityGroup"
          }
        ]
      }
    },
    "SSHSecurityGroup": {
      "Properties": {
        "GroupDescription": "Enable SSH access to EC2 hosts",
        "SecurityGroupIngress": [
          {
            "CidrIp": "8.8.8.8/32",
            "FromPort": 22,
            "IpProtocol": "tcp",
            "ToPort": 22
          }
        ]
      },
      "Type": "AWS::EC2::SecurityGroup"
    }
  }
}
```

### Custom AMIs

Another common customization is to provide different AMIs. Porter defaults to
using Amazon Linux AMI so for the best results, custom AMIs should be based on
Amazon Linux AMI.

Again, porter doesn't redefined the `AWS::AutoScaling::LaunchConfiguration`
resource, and because `ImageId` is defined, porter leaves it alone.

`.porter/cfn.json` for a single region

```json
{
  "Resources": {
    "AutoScalingLaunchConfiguration": {
      "Type": "AWS::AutoScaling::LaunchConfiguration",
      "Properties": {
        "ImageId": "ami-abcd1234"
      }
    }
  }
}
```

`.porter/cfn.json` for multi-region

```json
{
  "Mappings": {
    "RegionToCustomAMI": {
      "ap-northeast-1": {
        "Key": "ami-abcd1234"
      },
      "ap-northeast-2": {
        "Key": "ami-abcd1235"
      },
      "ap-southeast-1": {
        "Key": "ami-abcd1236"
      },
      "ap-southeast-2": {
        "Key": "ami-abcd1237"
      },
      "eu-central-1": {
        "Key": "ami-abcd1238"
      },
      "eu-west-1": {
        "Key": "ami-abcd1239"
      },
      "sa-east-1": {
        "Key": "ami-abcd1240"
      },
      "us-east-1": {
        "Key": "ami-abcd1241"
      },
      "us-west-1": {
        "Key": "ami-abcd1242"
      },
      "us-west-2": {
        "Key": "ami-abcd1243"
      }
    }
  },
  "Resources": {
    "AutoScalingLaunchConfiguration": {
      "Type": "AWS::AutoScaling::LaunchConfiguration",
      "Properties": {
        "ImageId": {
          "Fn::FindInMap": [
            "RegionToCustomAMI",
            {
              "Ref": "AWS::Region"
            },
            "Key"
          ]
        }
      }
    }
  }
}
```

### Additional EC2 permissions

A service often interacts with other AWS services and needs permission to do so.
Here is an example enabling EC2 instances to query a madeup DynamoDB table
called `books_table`

`.porter/cfn.json`

```json
{
  "Resources": {
    "EC2Role": {
      "Type": "AWS::IAM::Role",
      "Properties": {
        "Policies": [
          {
            "PolicyDocument": {
              "Statement": [
                {
                  "Resource": "arn:aws:dynamodb:us-east-1:123456789012:table/books_table",
                  "Action": [
                    "dynamodb:GetItem",
                    "dynamodb:Query"
                  ],
                  "Effect": "Allow"
                }
              ]
            }
          }
        ]
      }
    }
  }
}
```
