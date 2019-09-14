# go-nats
NAT type discovery tool using STUN written purely in Go, powered by [pion](pion.ly).

## Usage
```
$ go build
$ ./go-nats -h
Usage of ./go-nats:
  -s string
    	STUN server address. (defaults to "stun.ekiga.net:3478" (default "stun.ekiga.net:3478")
  -v	Verbose
```

Example:
```
$ ./go-nats -s stun.sipgate.net
{
  "isNatted": true,
  "mappingBehavior": 0,
  "filteringBehavior": 2,
  "portPreservation": true,
  "natType": "Port-restricted cone NAT",
  "externalIP": "23.3.5.241"
}
```

## Public STUN servers
STUN servers to use must support RFC 5780 (NAT Bhavior Discovery Using STUN).
Here's a list of public STUN servers that worked with go-nats as of Sep. 13, 2019.

* stun.ekiga.net
* stun.callwithus.com
* stun.counterpath.net
* stun.sipgate.net
* stun.sipgate.net:10000
* stun.1-voip.com

> TODO: there may be more from this list:
* [Emercoin/ENUMER projects](http://olegh.ftp.sh/public-stun.txt)
