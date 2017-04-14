#!/bin/bash -e

render_template() {
  eval "echo \"$(cat $1)\""
}

env
tree -a .

CONF=$(render_template .porter/config)
echo "$CONF"
echo "$CONF" > .porter/config

CIS_AMI_JSON=$(render_template cis_ami.json)
echo "$CIS_AMI_JSON"
echo "$CIS_AMI_JSON" > cis_ami.json

VPC_JSON=$(render_template vpc.json)
echo "$VPC_JSON"
echo "$VPC_JSON" > vpc.json
