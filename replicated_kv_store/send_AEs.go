package main

import (
	"context"
	"log"
	"sync/atomic"
	"time"

	"github.com/krithikvaidya/distributed-dns/replicated_kv_store/protos"
)

// To send AppendEntry to single replica, and retry if needed.
func (node *RaftNode) LeaderSendAE(replica_id int32, upper_index int32, client_obj protos.ConsensusServiceClient, msg *protos.AppendEntriesMessage) (status bool) {

	response, err := client_obj.AppendEntries(context.Background(), msg)

	if err != nil {
		// ...
	}

	if response.Success == false {

		if node.state != Leader {
			return
		}

		if response.Term > node.currentTerm {

			node.ToFollower(response.Term)
			return false
		}

		// will reach here if response.Term <= node.currentTerm and response.Success == false

		node.nextIndex[replica_id]--

		var entries []*protos.LogEntry

		for i := msg.PrevLogIndex; i <= upper_index; i++ {
			entries = append(entries, &node.log[i])
		}

		prevLogIndex := int32(msg.PrevLogIndex - 1)
		prevLogTerm := int32(-1)

		if prevLogIndex >= 0 {
			prevLogTerm = node.log[prevLogIndex].Term
		}

		new_msg := &protos.AppendEntriesMessage{

			Term:         node.currentTerm,
			LeaderId:     node.replica_id,
			PrevLogIndex: prevLogIndex,
			PrevLogTerm:  prevLogTerm,
			LeaderCommit: node.commitIndex,
			Entries:      entries,
		}

		return node.LeaderSendAE(replica_id, upper_index, client_obj, new_msg)

	} else {

		node.nextIndex[replica_id] = upper_index + 1
		node.matchIndex[replica_id] = upper_index

		return true

	}

}

// Leader sending AppendEntries to all other replicas.
func (node *RaftNode) LeaderSendAEs(msg_type string, msg *protos.AppendEntriesMessage, upper_index int32) {

	replica_id := int32(0)

	// node.raft_node_mutex.RLock() // for node.peer_replica_clients.
	// defer node.raft_node_mutex.RUnlock()
	// (The above are not required, since in the current implementation it doesnt change after initialization)

	successes := int32(1)

	for _, client_obj := range node.peer_replica_clients {

		if replica_id == node.replica_id {
			replica_id++
			continue
		}

		go func(node *RaftNode, client_obj protos.ConsensusServiceClient, replica_id int32, upper_index int32) {

			node.raft_node_mutex.Lock()
			// log.Printf("\nLocked in LeaderSendAEs spawned goroutine\n")
			// log.Printf("\nDATA: \n")
			// log.Printf("\nreplica_id: %v, upper_index: %v, msg: %v\n", replica_id, upper_index, msg)
			if node.LeaderSendAE(replica_id, upper_index, client_obj, msg) {
				tot_success := atomic.AddInt32(&successes, 1)
				if (msg_type != "HBEAT") && (tot_success > (node.n_replicas)/2) {
					log.Printf("node.commit_chan true")
					// node.commit_chan <- true
				}
			} else {
				log.Printf("node.commit_chan false")
				// node.commit_chan <- false
			}

			// log.Printf("\nUnLocked in LeaderSendAEs spawned goroutine\n")
			node.raft_node_mutex.Unlock()

		}(node, client_obj, replica_id, upper_index)

		replica_id++

	}

}

//HeartBeats is a goroutine that periodically makes leader
//send heartbeats as long as it is the leader
func (node *RaftNode) HeartBeats() {

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {

		<-ticker.C

		node.raft_node_mutex.RLock()
		// log.Printf("\nRLock in HeartBeats\n")

		if node.state != Leader {

			// log.Printf("\nRunLock in HeartBeats\n")
			node.raft_node_mutex.RUnlock()
			return
		}

		replica_id := 0

		prevLogIndex := node.nextIndex[replica_id] - 1
		prevLogTerm := int32(-1)

		if prevLogIndex >= 0 {
			prevLogTerm = node.log[prevLogIndex].Term
		}

		// send heartbeat
		var entries []*protos.LogEntry

		hbeat_msg := &protos.AppendEntriesMessage{

			Term:         node.currentTerm,
			LeaderId:     node.replica_id,
			PrevLogIndex: prevLogIndex,
			PrevLogTerm:  prevLogTerm,
			LeaderCommit: node.commitIndex,
			Entries:      entries,
		}

		node.LeaderSendAEs("HBEAT", hbeat_msg, int32(len(node.log)))
		// log.Printf("\nRunLock in HeartBeats\n")
		node.raft_node_mutex.RUnlock()

	}
}
