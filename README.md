# Strongly Consistent DNS using Raft Consensus

[![Go Report Card](https://goreportcard.com/badge/github.com/krithikvaidya/distributed-dns)](https://goreportcard.com/report/github.com/krithikvaidya/distributed-dns)
[![](https://godoc.org/github.com/krithikvaidya/distributed-dns?status.svg)](https://godoc.org/github.com/krithikvaidya/distributed-dns)

A repository containing our learnings and implementations for the project "Distributed DNS in the Cloud" under IEEE-NITK.

- [Blog Article](https://ieee.nitk.ac.in/virtual-expo/raft-based-dns/) explaining the project.

- Material Related to Learning Phase - [Repo](https://github.com/krithikvaidya/distdns-learning)

- Branches: 
  - [main](https://github.com/krithikvaidya/distributed-dns) - replicated key-value store using Raft consensus (and tests)
  - [aws-dns](https://github.com/krithikvaidya/distributed-dns/tree/aws-dns) (current branch) - extends the replicated key-value store so that it can be used in the DNS service

- [Wiki](https://github.com/krithikvaidya/distributed-dns/wiki) explaining different aspects of the Raft consensus implementation.
