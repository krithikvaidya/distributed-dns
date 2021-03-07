package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/krithikvaidya/distributed-dns/replicated_kv_store/protos"
)

type testing_st struct {
	mu        sync.Mutex  // The mutex for performing operations on the struct
	t         *testing.T  // The testing object for utility funcs to display errors
	n         int         // The number of nodes in the system
	rep_addrs []string    // The addresses of the replicas in the system
	nodes     []*RaftNode // The RaftNode objects of the individual replicas
	active    []bool      // The status of each node, whether it is active(true) or not(false)
	start     time.Time   // Time at which make_testing_st() was called
}

/*
 * This function is initially called in a test case for
 * setting up a system on which raft can be run.
 *
 * It initializes a struct for managing the current
 * testable raft system.
 *
 * The system is created by calling two functions,
 * `setup_raft_node()` for initial creation of the
 * `RaftNode` structs and `connect_raft_nodes()` to
 * start the required services and connect the nodes
 * together.
 */
func start_test(t *testing.T, n int) *testing_st {
	new_test_st := &testing_st{}
	new_test_st.t = t
	new_test_st.n = n
	new_test_st.nodes = make([]*RaftNode, n)
	new_test_st.active = make([]bool, n)
	new_test_st.rep_addrs = make([]string, n)

	// Create the replicas
	for i := 0; i < n; i++ {

		// Create the context for the current node
		master_ctx, master_cancel := context.WithCancel(context.Background())

		// Obtain the RaftNode object for the current node
		new_test_st.nodes[i] = setup_raft_node(master_ctx, i, new_test_st.n, true)

		// Set the master context and cancel entities in the node metadata struct
		new_test_st.nodes[i].meta.master_ctx = master_ctx
		new_test_st.nodes[i].meta.master_cancel = master_cancel

		new_test_st.rep_addrs[i] = ":500" + strconv.Itoa(i)
	}

	// Connect the replicas together to setup the system
	for i := 0; i < n; i++ {

		// Connect the current node to the system
		go new_test_st.nodes[i].connect_raft_node(new_test_st.nodes[i].meta.master_ctx, i, new_test_st.rep_addrs, true)

		// Set the current node as active
		new_test_st.active[i] = true
	}

	return new_test_st
}

/*
 * This function is used to end a test case.
 *
 * This is done by bringing down all nodes in the current testing
 * system by canceling the contexts of all of the nodes.
 *
 * Needs to be called at the end of every test.
 */
func end_test(test_st *testing_st) {
	for i := 0; i < test_st.n; i++ {
		test_st.nodes[i].meta.master_cancel()
	}
}

/*
 * This function is used to simulate the crash of a node
 * in the current system by disconnecting it with other
 * nodes and then killing the various services on the
 * node.
 */
func (test_st *testing_st) crash_raft_node(id int) {
	// Save the current status in persistent storage
	test_st.nodes[id].persistToStorage()

	// Mark the node as 'inactive'
	test_st.active[id] = false

	// Cancel the context of the node to be crashed
	test_st.nodes[id].meta.master_cancel()
}

/*
 * This function can be called to find the number of
 * active nodes with the states as 'Leader'
 */
func (test_st *testing_st) count_leader() int {
	n_leaders := 0
	for i := 0; i < test_st.n; i++ {
		if test_st.active[i] {
			if test_st.nodes[i].state == Leader {
				n_leaders++
			}
		}
	}

	return n_leaders
}

/*
 * This function is used to find the replica ID of the
 * current leader node in the system.
 */
func (test_st *testing_st) find_leader() int {
	leader_id := -1
	for i := 0; i < test_st.n; i++ {
		if test_st.active[i] {
			if test_st.nodes[i].state == Leader {
				leader_id = i
			}
		}
	}
	return leader_id
}

/*
 * This function checks whether the terms on the active
 * nodes are same.
 */
func (test_st *testing_st) check_terms() bool {
	var sys_term int32
	sys_term = -1
	for i := 0; i < test_st.n; i++ {
		if test_st.active[i] {
			if sys_term == -1 {
				sys_term = test_st.nodes[i].currentTerm
			} else if sys_term != test_st.nodes[i].currentTerm {
				return false
			}
		}
	}

	return true
}

/*
 * This function checks if, at a particular index, the majority of
 * the nodes have the same log entry.
 */
func (test_st *testing_st) check_consensus(index int) int {
	count := 0

	var logs []protos.LogEntry

	// Obtain the log values for the given index from all the currently active
	// nodes
	for i := 0; i < test_st.n; i++ {
		if test_st.active[i] == true {
			logs = append(logs, test_st.nodes[i].log[index])
		}
	}

	// Check the most frequent element of the array
	for i := 0; i < test_st.n; i++ {
		curr_count := 1
		for j := 0; j < test_st.n; j++ {
			// Ignore the current index
			if i == j {
				continue
			}

			// Check the entry at the current index against the other indexes
			if logs[i].Term == logs[j].Term &&
				len(logs[i].Operation) == len(logs[j].Operation) &&
				logs[i].Clientid == logs[j].Clientid {
				length := len(logs[i].Operation)
				agree := true
				for k := 0; k < length; k++ {
					if logs[i].Operation[k] != logs[j].Operation[k] {
						agree = false
					}
				}
				if agree == true {
					curr_count++
				}
			}
		}

		// Taking only the max value
		if curr_count > count {
			count = curr_count
		}
	}

	return count
}

func (test_st *testing_st) restartRaftNode(id int) {

	// Create the context for the node
	master_ctx, master_cancel := context.WithCancel(context.Background())

	// Obtain the RaftNode object for the node
	test_st.nodes[id] = setup_raft_node(master_ctx, id, test_st.n, true)

	// Set the master context and cancel entities in the node metadata struct
	test_st.nodes[id].meta.master_ctx = master_ctx
	test_st.nodes[id].meta.master_cancel = master_cancel

	test_st.rep_addrs[id] = ":500" + strconv.Itoa(id)

	// Connect the node to the system
	go test_st.nodes[id].connect_raft_node(test_st.nodes[id].meta.master_ctx, id, test_st.rep_addrs, true)

	// Set the node as active
	test_st.active[id] = true
}

func (test_st *testing_st) addNewLogEntries(n int, t *testing.T, startIndex int) {
	leader_id := test_st.find_leader()

	if leader_id == -1 {
		t.Errorf("Invalid leader ID %v", leader_id)
	}
	//get leader address
	leader_addr := ":400" + strconv.Itoa(leader_id)

	for i := startIndex; i < n+startIndex; i++ {

		url := fmt.Sprintf("http://localhost%s/key%v", leader_addr, i)
		body := strings.NewReader(fmt.Sprintf("value=value%v&client=admin", i))
		req, err := http.NewRequest("POST", url, body)
		if err != nil {
			t.Errorf("Error in making request %v", err)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Errorf("Error in response %v", err)
		}
		resp.Body.Close()

		// Allow entries to be added
		time.Sleep(2 * time.Second)
	}
}
