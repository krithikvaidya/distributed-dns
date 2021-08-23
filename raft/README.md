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
