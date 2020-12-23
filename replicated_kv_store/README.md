# Replicated Key Value Store using Raft Consensus

### How to Run:

- Clone the repo.

- Run multiple terminals (one for each replica).

- Change directory to this folder (replicated_kv_store) in each terminal window.

- On each terminal, run ```go run . -n <number_of_replicas>``` and follow the on screen instructions.

- For running tests, use ```go test```.

### Instructions for writing new test cases:

- A **test file** is a single file with a collection of **test cases**.

- The test file should be named with a desired filename followed by `_test.go`, like `yourfilename_test.go`.

- Each test case is a function with its name beginning with the word `Test` followed by a word or a phrase that starts with a capital letter. For example `TestRequestVote` and `TestRaft` are valid test case names while `Testrequestvote`, `TestrequestVote` and `Testraft` are invalid test case names.

- The first and only parameter for every test case function is `t *testing.T`.

- For handling failure conditions, call `t.Error()` or `t.Fail()`. These are similar to `fmt.Print()`. Use `fmt.Errorf()` if you want formatting similar to `fmt.Printf()`.

- `t.Log()` can be used to provide debug information.

- Check the test case written in the file `raft_node_test.go` for reference.

- For keeping tests organized, it is preferable to have one test file per source file. For example, all the tests related to the functions in `raft_node.go` should be written in the test file `raft_node_test.go`.

- Please write proper documentation for the tests, describing what feature it tests and how it manages to do so. This can be written in a comment block before the corresponding test case function.