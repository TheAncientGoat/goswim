# Go Swim
Very basic implementation of a SWIM protocol that can keep track of a set of peers by gossip

## Limitations
* IPv4 only
* Non-safe map mutation
* Innefficient string based protocol

## Running

Usage of ./goswim:
  -ip string
        IP of server (default "0.0.0.0")
  -peerIp string
        IP of server to connect to (default "0.0.0.0")
  -peerPort int
        Port of the initial peer to connect to (default 6969)
  -port int
        Port to listen on (default 6969)
