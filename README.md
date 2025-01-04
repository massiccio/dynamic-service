# Rationale

Modern enterprise applications have numerous moving parts that need continual upkeep, including:
* TLS certificate management and termination
* Servers setup and maintenance
* API gateway integration
* Load balancing
* Infratructure scaling for both capacity and maintenance purposes
* Policies enforcement
* Retries and circuit breaker
* etc.

This project is not trying to solve all of the above. Instead, it aims at exploring how to automatically tackle some of the above challenges by leveraging the [Consul](https://www.consul.io/) dynamic service discovery capabilities for service registration and DNS, [Caddy](https://caddyserver.com/) for TLS termination, and [KrakenD](https://www.krakend.io/open-source/) as an API Gateway.


![Architecture](/docs/diagram.png)

# Requirements

This repository assumes access to:
* Go
* curl
* Docker

# Running the Deployment Locally

```bash
$ docker compose up
[+] Running 3/3
 ✔ krakend Pulled                                                                                                                          1.3s 
 ✔ caddy Pulled                                                                                                                            1.3s 
 ✔ consul Pulled                                                                                                                           1.3s 
[+] Running 5/5
 ✔ Network dynamic-app_vpcbr             Created                                                                                           0.1s 
 ✔ Container consul                      Created                                                                                           0.6s 
 ✔ Container caddy                       Created                                                                                           0.7s 
 ✔ Container dynamic-app-ping-service-1  Created                                                                                           0.5s 
 ✔ Container krakend                     Created                                                                                           0.7s 
Attaching to caddy, consul, ping-service-1, krakend
```

# Backend Server

The application server ("backend", using KrakenD's terminology) executes the defined business logic, responding to user demand. While in a more realistinc scenario it may interact with databases or other services, in this project it [implements](https://github.com/massiccio/dynamic-service/blob/master/app/main.go) two endpoints, /ping and /pong, which return a JSON structure that also include the details of the backend executing the request.


# Endpoint Redirection

Caddy acts as an enterprise-grade web server with automatic HTTPS.

When hitting the HTTP endpoint, Caddy will automatically [redirect](https://github.com/massiccio/dynamic-service/blob/master/services/caddy/Caddyfile#L16) user requests to the HTTPS endpoint - note the ["-L" flag](https://curl.se/docs/manpage.html#-L) when executing the command, which tells curl to follow the redirect.

```bash
$ curl -L -i -k http://localhost:8080/ping
HTTP/1.1 308 Permanent Redirect
Location: https://localhost:8443/ping
Server: Caddy
Date: Sat, 04 Jan 2025 10:28:23 GMT
Content-Length: 0

HTTP/2 200
alt-svc: h3=":443"; ma=2592000
cache-control: public, max-age=300
content-type: application/json; charset=utf-8
date: Sat, 04 Jan 2025 10:28:23 GMT
server: Caddy
x-krakend: Version 2.8.0
x-krakend-completed: true
content-length: 83

{"message":"Pong!","serviceID":"ping-service-8f55cb0c00be","serviceIP":"192.0.2.3"}
```


The "pong" endpoint fails with HTTP status code 500 with [probability 1/3](https://github.com/massiccio/dynamic-service/blob/master/app/main.go#L58).

Successful request:
```bash
$ curl -i -k https://localhost:8443/pong
HTTP/2 200
alt-svc: h3=":443"; ma=2592000
cache-control: public, max-age=300
content-type: application/json; charset=utf-8
date: Sat, 04 Jan 2025 10:32:48 GMT
server: Caddy
x-krakend: Version 2.8.0
x-krakend-completed: true
content-length: 83

{"message":"Ping!","serviceID":"ping-service-8f55cb0c00be","serviceIP":"192.0.2.3"}
```

Failed request:
```bash
$ curl -i -k https://localhost:8443/pong
HTTP/2 500
alt-svc: h3=":443"; ma=2592000
date: Sat, 04 Jan 2025 10:32:50 GMT
server: Caddy
x-krakend: Version 2.8.0
x-krakend-completed: false
content-length: 0
```

# API Composition

A really nice feature of KrakenD is API [composition and aggregation](https://www.krakend.io/docs/endpoints/response-manipulation/). By declaring a [custom endpoint with two backends](https://github.com/massiccio/dynamic-service/blob/master/services/krakend/krakend.json#L48), KrakenD will automatically aggregate and merge from the backends, while by using the _group_ keyword, our API Gateway creates a new key and encapsulates the response.


```bash
$ curl -k https://localhost:8443/ping-pong
{
    "ping":{"ping-message":"Pong!","ping-serviceID":"ping-service-2fc167a09f2b","ping-serviceIP":"192.0.2.2"},
    "pong":{"pong-message":"Ping!","pong-serviceID":"ping-service-2fc167a09f2b","pong-serviceIP":"192.0.2.2"}
}
```


# Dynamic Service Discovery

The backend configuration is dynamic - the REST services [register](https://github.com/massiccio/dynamic-service/blob/master/app/consul_utils.go#L43) themselves with Consul upon startup, while Krakend uses the DNS information it retrieves from Consul to forward the requests to the backend.
Consul periodically executes health checks on all services, automatically removing failed backends from its DNS records. For more details on how to set up health checks, refer to the Consul [documentation](https://developer.hashicorp.com/consul/docs/services/usage/checks).


Consul is exposed on port 8500, so we can verify its status at http://localhost:8500/ui/dc1/services.
In particular we can gether some details about the ping-service at http://localhost:8500/ui/dc1/services/ping-service/instances


![Architecture](/docs/ping-service-1.png)

```bash
curl -k https://localhost:8443/ping
{"message":"Pong!","serviceID":"ping-service-c0a72cab0b17","serviceIP":"192.0.2.3"}m
```

Next, we increase the number of endpoints from 1 to 2

```bash
docker compose scale ping-service=2
[+] Running 3/3
 ✔ Container dynamic-app-ping-service-1  Started                                                                                                      12.9s
 ✔ Container consul                      Started                                                                                                       1.7s
 ✔ Container dynamic-app-ping-service-2  Started                                                                                                       1.3s
```

![Architecture](/docs/ping-service-2.png)

If we execute the same request as above, our request may get routed to a different backend:
```bash
$ curl -k https://localhost:8443/ping
{"message":"Pong!","serviceID":"ping-service-e750931c6d0e","serviceIP":"192.0.2.3"}
$ curl -k https://localhost:8443/ping
{"message":"Pong!","serviceID":"ping-service-c0a72cab0b17","serviceIP":"192.0.2.5"}
```

Note that we didn't have to change any configuration after adding the extra backend.