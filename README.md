# Distributed DNS

[![](https://godoc.org/github.com/krithikvaidya/distributed-dns?status.svg)](https://godoc.org/github.com/krithikvaidya/distributed-dns)

A repository containing our learnings and implementations for the project "Distributed DNS in the Cloud" under IEEE-NITK (Work in Progress)

Material Related to Learning Phase - [Repo](https://github.com/krithikvaidya/distdns-learning)

Branches: [main](https://github.com/krithikvaidya/distributed-dns), [aws-dns](https://github.com/krithikvaidya/distributed-dns/tree/aws-dns)

[Wiki](https://github.com/krithikvaidya/distributed-dns/wiki)

# Strongly Consistent DNS using Raft Consensus

## How to Run Locally (As a Replicated Key-Value Store):

- Clone the repo.

- Run multiple terminals (one for each replica).

- Change directory to the cloned repo in each terminal window.

- On each terminal, run ```go run . -n <number_of_replicas>``` and follow the on screen instructions.

- For running tests, use ```go test```. Preferably use ```go test -p 1``` to prevent parallel execution.

## Making requests from the client:

Assume that the leader is running the server listening for client requests on port :xyzw on localhost.

Use curl to access the key value store and perform any of the CRUD functions<br>

POST request : ```curl -d "value=<value>&client=<id>" -X POST http://localhost:xyzw/<key>```<br>
GET request : ```curl -X GET  http://localhost:xyzw/<key>```<br>
PUT request : ```curl -d "value=<value>&client=<id>" -X PUT http://localhost:xyzw/<key>```<br>
DELETE request : ```curl -X DELETE  http://localhost:xyzw/<key>```<br>

## Using as a Distributed DNS in the Cloud

- (Instructions to be added)

## General instructions for testing:

- A **test file** is a single file with a collection of **test cases**.

- The test file should be named with a desired filename followed by `_test.go`, like `yourfilename_test.go`.

- Each test case is a function with its name beginning with the word `Test` followed by a word or a phrase that starts with a capital letter. For example `TestRequestVote` and `TestRaft` are valid test case names while `Testrequestvote`, `TestrequestVote` and `Testraft` are invalid test case names.

- The first and only parameter for every test case function is `t *testing.T`.

- For handling failure conditions, call `t.Error()` or `t.Fail()`. These are similar to `fmt.Print()`. Use `fmt.Errorf()` if you want formatting similar to `fmt.Printf()`.

- `t.Log()` can be used to provide debug information.

- Check the test cases written in the file `raft_test.go` for reference.

- Please write proper documentation for the tests, describing what feature it tests and how it manages to do so. This can be written in a comment block before the corresponding test case function.

## How to write new test cases:

- The functions in the file `testing_utils.go`, should contain the necessary functions needed for testing, for example, functions for setting up a Raft system, crashing a single node, destroying the whole system, counting number of leaders, etc.

- Always start a test by calling the `start_test()` function which should return a pointer to the testing struct of type `testing_st`. The parameters to `start_test` are the `t *testing.T` object and the number of nodes that should exist in the Raft system to be created.

- `testing_st` is a struct that needs to be created for every test case. This will hold the `RaftNode` structs and other data related to individual nodes so that a global view of the system is available.

- Perform operations on the testing system and check its status using the functions in `testing_utils.go`.

- End the test case by using `end_test()` and passing the testing system to it as a parameter.

