# Strongly Consistent DNS using Raft Consensus

[![Go Report Card](https://goreportcard.com/badge/github.com/krithikvaidya/distributed-dns)](https://goreportcard.com/report/github.com/krithikvaidya/distributed-dns)
[![](https://godoc.org/github.com/krithikvaidya/distributed-dns?status.svg)](https://godoc.org/github.com/krithikvaidya/distributed-dns)

A repository containing our learnings and implementations for the project "Distributed DNS in the Cloud" under IEEE-NITK.

- [Blog Article](https://ieee.nitk.ac.in/virtual-expo/raft-based-dns/) explaining the project.

- Material Related to Learning Phase - [Repo](https://github.com/krithikvaidya/distdns-learning)

- Branches: 
  - [main](https://github.com/krithikvaidya/distributed-dns) (current branch) - replicated key-value store using Raft consensus (and tests)
  - [aws-dns](https://github.com/krithikvaidya/distributed-dns/tree/aws-dns) - extends the replicated key-value store so that it can be used in the DNS service

- [Wiki](https://github.com/krithikvaidya/distributed-dns/wiki) explaining different aspects of the Raft consensus implementation.

## How to Run Locally (As a Replicated Key-Value Store):

- Clone the repo.

- Run multiple terminals (one for each replica).

- Change directory to the cloned repo in each terminal window.

- On each terminal, run ```go run . -n <number_of_replicas>``` and follow the on screen instructions.

- For running tests, use ```go test```. **NOTE**: Do not run the tests in parallel.

## Making requests from the client:

Assume that the leader is running the server listening for client requests on port :xyzw on localhost.

Use curl to access the key value store and perform any of the CRUD functions<br>

POST request : ```curl -d "value=<value>&client=<id>" -X POST http://localhost:xyzw/<key>```<br>
GET request : ```curl -X GET  http://localhost:xyzw/<key>```<br>
PUT request : ```curl -d "value=<value>&client=<id>" -X PUT http://localhost:xyzw/<key>```<br>
DELETE request : ```curl -X DELETE  http://localhost:xyzw/<key>```<br>

