porterd
=======

porterd is a HTTP service that runs on your EC2 instance

Connecting to porterd
---------------------

The following environment variables are available in your service's docker
container.

```
PORTERD_TCP_ADDR
PORTERD_TCP_PORT
```

This command from *inside* the container (not on the host) should return a 200
response
```
curl -i "http://$PORTERD_TCP_ADDR:$PORTERD_TCP_PORT/health"
```

Logging
-------

porterd generates a request id for every request. The key is `RequestId` and
the value is a
[UUIDv4](https://en.wikipedia.org/wiki/Universally_unique_identifier#Version_4_.28random.29)

Both the key and value can be overridden using headers.

To just override the value with your own request id use `X-Request-Id`

To override the key and value use `X-Request-Id-Key` and `X-Request-Id-Value`
