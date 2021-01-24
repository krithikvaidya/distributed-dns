package main

import (
	"testing"
	"time"
)

/*
 * This test case checks whether the initial election works as intended
 *
 * The procedure is:
 * 1. It sets up the system by creating a set of nodes. This creation of
 * nodes also involves starting the required services on the individual machine.
 * 2. These nodes are then connected to each other through the procedures
 * implemented in the master application.
 * 3. After this connection has been set up, the function `count_leader()` is
 * used to count the number of leaders in the current system, after the first
 * election.
 * 4. The expected number of leaders for the system at this point is 1, and the
 * test passes if this is the count.
 *
 * Here, the first two steps are taken care of, through the creation of the
 * testing struct `testing_st`. This is done using the function `make_testing_st`,
 * which should be run at the beginning of every test.
 */
func TestInitialElection(t *testing.T) {
	n := 5
	test_sys := make_testing_st(t, n)
	time.Sleep(3 * time.Second)

	no_of_leaders := test_sys.count_leader()
	if no_of_leaders != 1 {
		t.Errorf("Invalid number of leaders %v, expected 1", no_of_leaders)
	}
}