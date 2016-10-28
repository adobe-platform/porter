#!/bin/bash

echo "DEV_MODE $DEV_MODE" 1>&2

if [[ $PORTER_ENVIRONMENT = 'CustomVPC' ]]; then

cat <<'EOF'
SECRET=$(porter_get_secrets)
[[ $SECRET = 'hi' ]] || exit 1

curl -Lo /usr/bin/jq https://github.com/stedolan/jq/releases/download/jq-1.5/jq-linux64
chmod 555 /usr/bin/jq

cat <<'HOTSWAP_SIGNAL' > /usr/bin/porter_hotswap_signal
#!/bin/bash

read STDIN
HOST_PORTS=`echo $STDIN | jq -r '.containers[].hostPort'`
for HOST_PORT in $HOST_PORTS; do

cat <<'DOCKERFILE' | docker build -t tcpdump-$HOST_PORT -
FROM ubuntu:16.04

RUN apt-get update
RUN apt-get install -y tcpdump

CMD tcpdump -w /host/tcpdump-$PORT -n -s 100 -i lo port $PORT
DOCKERFILE

docker run -d \
-v /home/ec2-user:/host \
--net=host \
--privileged \
-e PORT=$HOST_PORT \
tcpdump-$HOST_PORT
done

HOTSWAP_SIGNAL
chmod 755 /usr/bin/porter_hotswap_signal
EOF

fi
