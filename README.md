Round-Robin and Least-Connected all work by connecting a client's request to a potentially different server, there are never any guarantees that a client will be directed to the same server. 

IP Hashing load balacning uses an IP address as a hashing key to determine what server a server group should be selected for. Same requests from the same client are always mapped to the same server. 


use the HTTP Header and a hashing library like hash/fnv. 