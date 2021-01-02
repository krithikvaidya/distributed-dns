package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/krithikvaidya/distributed-dns/replicated_kv_store/protos"
)

//WriteCommand allows clients to submit a new command to the leader
func (node *RaftNode) WriteCommand(operation []string) bool {
	node.raft_node_mutex.Lock()
	defer node.raft_node_mutex.Unlock()

	// True only if leader
	if node.state == Leader {
		//append to local log
		node.log = append(node.log, protos.LogEntry{Term: node.currentTerm, Operation: operation})

		var entries []*protos.LogEntry
		entries = append(entries, &node.log[len(node.log)-1])

		msg := &protos.AppendEntriesMessage{

			Term:         node.currentTerm,
			LeaderId:     node.replica_id,
			PrevLogIndex: int32(len(node.log) - 1),
			PrevLogTerm:  node.log[len(node.log)-1].Term,
			LeaderCommit: node.commitIndex,
			Entries:      entries,
		}

		node.LeaderSendAEs(operation[0], msg, int32(len(node.log)-1))

		val := <-node.successfulwrite //Written to from AE when majority of nodes have replicated the write

		return val
	}

	return false
}

// ReadCommand is different since read operations do not need to be added to log
func (node *RaftNode) ReadCommand(key int) bool {

	node.raft_node_mutex.RLock()
	defer node.raft_node_mutex.RUnlock()

	node.StaleReadCheck()

	if node.state == Leader {

		// assuming that if an operation on the state machine succeeds on one of the replicas,
		// it will succeed on all. and vice versa.
		url := fmt.Sprintf("http://localhost%s/%d", node.kvstore_addr, key)

		resp, err := http.Get(url)

		if err != nil {
			log.Printf("\nREAD successful. \nData: %v\n", resp)
		} else {
			log.Printf("\nREAD failed. \nError: %v\n", err)
		}

		return true
	}

	return false
}

// StaleReadCheck sends dummy heartbeats to make sure that a new leader has not come
func (node *RaftNode) StaleReadCheck() {
	replica_id := 0

	var entries []*protos.LogEntry

	hbeat_msg := &protos.AppendEntriesMessage{

		Term:         node.currentTerm,
		LeaderId:     node.replica_id,
		PrevLogIndex: node.nextIndex[replica_id] - 1,
		PrevLogTerm:  node.log[node.nextIndex[replica_id]-1].Term,
		LeaderCommit: node.commitIndex,
		Entries:      entries,
	}

	node.LeaderSendAEs("HBEAT", hbeat_msg, int32(len(node.log)))
}
