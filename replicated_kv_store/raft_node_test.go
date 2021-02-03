package main

import (
	"bytes"
	"fmt"
	"testing"
)

/*
 * This test is for the function InitializeNode() in `raft_node.go`
 *
 * Here, we call `InitializeNode()` and check whether each value in
 * the `RaftNode` struct obtained from the function has the expected
 * value and the type.
 */
func TestInitializeNode(t *testing.T) {
	/*
	 * Arbitrarily chosen values for number of replicas and
	 * the replica ID
	 */
	var no_of_replicas, replica_id int32
	var replica_key_value_port string
	no_of_replicas = 5
	replica_id = 1
	replica_key_value_port = ":8081"
	node := InitializeNode(no_of_replicas, int(replica_id), replica_key_value_port)

	// Here, we check every member of the RaftNode struct

	// Check n_replicas
	if node_meta.n_replicas != no_of_replicas {
		t.Errorf("Invalid value for n_replicas, expected %v", no_of_replicas)
	}

	// Check the type of `ready_chan`
	var buf1 bytes.Buffer
	fmt.Fprintf(&buf1, "%T", node.ready_chan)
	ready_chan_type := buf1.String()

	if ready_chan_type != "chan bool" {
		t.Errorf("Invalid type %v for ready_chan, expected %v", ready_chan_type, "chan_bool")
	}

	// Check the value of `replicas_ready`
	if node.replicas_ready != 0 {
		t.Errorf("Invalid value %v for replicas_ready, expected 0", node.replicas_ready)
	}

	// Check the value of `replica_id`
	if node_meta.replica_id != replica_id {
		t.Errorf("Invalid value for replica_id, expected %v", replica_id)
	}

	// Check the type of `peer_replica_clients`
	var buf2 bytes.Buffer
	fmt.Fprintf(&buf2, "%T", node_meta.peer_replica_clients)
	peer_replica_clients_type := buf2.String()

	if peer_replica_clients_type != "[]protos.ConsensusServiceClient" {
		t.Errorf("Invalid type %v for peer_replica_clients, expected %v", peer_replica_clients_type, "[]protos.ConsensusServiceClient")
	}

	// Check the value of `state`
	if node.state != Follower {
		t.Errorf("Invalid value %v for state, expected %v", node.state, "Follower")
	}

	// Check the value of `currentTerm`
	if node.currentTerm != 0 {
		t.Errorf("Invalid value %v for currentTerm, expected 0", node.currentTerm)
	}

	// Check the value of `votedFor`
	if node.votedFor != -1 {
		t.Errorf("Invalid value %v for votedFor, expected -1", node.votedFor)
	}

	// Check the type of `stopElectiontimer`
	var buf3 bytes.Buffer
	fmt.Fprintf(&buf3, "%T", node.stopElectiontimer)
	stopElectiontimer_type := buf3.String()

	if stopElectiontimer_type != "chan bool" {
		t.Errorf("Invalid type %v for stopElectiontimer, expected %v", stopElectiontimer_type, "chan bool")
	}

	// Check the type of `electionResetEvent`
	var buf4 bytes.Buffer
	fmt.Fprintf(&buf4, "%T", node.electionResetEvent)
	electionResetEvent_type := buf4.String()

	if electionResetEvent_type != "chan bool" {
		t.Errorf("Invalid type %v for electionResetEvent, expected %v", electionResetEvent_type, "chan bool")
	}

	// Check the value of `commitIndex`
	if node.commitIndex != -1 {
		t.Errorf("Invalid value %v for commitIndex, expected 0", node.commitIndex)
	}

	// Check the value of `lastApplied`
	if node.lastApplied != -1 {
		t.Errorf("Invalid value %v for lastApplied, expected 0", node.lastApplied)
	}

	// Check the value of `kvstore_addr`
	if node_meta.kvstore_addr != ":8081" {
		t.Errorf("Invalid value %v for kvstore_addr, expected :8081", node_meta.kvstore_addr)
	}
}
