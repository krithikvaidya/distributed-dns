package main

import (
	"testing"
	"time"
)

/*
 * This test case checks whether the initial election works as intended
 *
 * The procedure is:
 * 1. It sets up the system by calling `start_test()`
 * 2. After this system has been set up, the function `count_leader()` is
 * used to count the number of leaders in the current system, after the first
 * election.
 * 3. The expected number of leaders for the system at this point is 1, and the
 * test passes if this is the count.
 */
func TestInitialElection(t *testing.T) {
	n := 5
	test_sys := start_test(t, n)

	// Allow time for initial election
	time.Sleep(5 * time.Second)

	// Check the number of leaders
	no_of_leaders := test_sys.count_leader()
	if no_of_leaders != 1 {
		t.Errorf("Invalid number of leaders %v, expected 1", no_of_leaders)
	}

	end_test(test_sys)
}

/*
 * This test case checks whether the re-election works as intended.
 * Re-election happens when the current leader node goes down and the remaining
 * nodes conduct an election to choose another leader.
 *
 * The procedure is:
 * 1. It sets up the system by calling `start_test()`
 * 2. After this connection has been set up, the function `find_leader()` is
 * used to find the current leader in the system, after the first election.
 * 3. The leader, if exists, is crashed using `crash_raft_node()`.
 * 4. After a few seconds, the number of leaders in the system is checked
 * using `count_leader()`. If it is 1, test passes.
 */
 func TestReElection(t *testing.T) {
	n := 5
	test_sys := start_test(t, n)

	// Allow time for initial election
	time.Sleep(5 * time.Second)

	// Find the current leader
	curr_leader_id := test_sys.find_leader()

	// If a leader exists, crash that node
	if curr_leader_id != -1 {
		test_sys.crash_raft_node(curr_leader_id)
	} else {
		t.Errorf("Invalid leader ID %v", curr_leader_id)
	}

	// Allow time for re-election
	time.Sleep(5 * time.Second)

	// Check number of leaders after re-election
	no_of_leaders := test_sys.count_leader()
	if no_of_leaders != 1 {
		t.Errorf("Invalid number of leaders %v, expected 1", no_of_leaders)
	}

	end_test(test_sys)
}
