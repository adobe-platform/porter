#!/bin/bash -e

porter build pack
porter build provision -e private
mv .porter-tmp/provision_receipt.json .porter-tmp/provision_receipt.json2
porter build provision -e private

while [[ 1 ]]; do
	porter build promote --elb live --provision-receipt .porter-tmp/provision_receipt.json
	porter build promote --elb live --provision-receipt .porter-tmp/provision_receipt.json2
done
