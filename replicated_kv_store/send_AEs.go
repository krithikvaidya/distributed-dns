package main

import (
	"context"
	"log"
	"sync/atomic"
	"time"

	"github.com/krithikvaidya/distributed-dns/replicated_kv_store/protos"
	"google.golang.org/grpc/peer"
)

// To send AppendEntry to single replica, and retry if needed (called by LeaderSendAEs defined below).
func (node *RaftNode) LeaderSendAE(replica_id int32, upper_index int32, client_obj protos.ConsensusServiceClient, msg *protos.AppendEntriesMessage) (status bool) {

	var response *protos.AppendEntriesResponse
	var err error

	//log.Printf("IN %d\n", replica_id)
	// Call the AppendEntries RPC for the given client
	var ctx context.Context
	response, err = client_obj.AppendEntries(peer.NewContext(ctx, &peer.Peer{}), msg)

	if err != nil {
		return false
	}

	if response.Success == false {

		if node.state != Leader {
			return false
		}

		if response.Term > node.currentTerm {

			node.ToFollower(response.Term)
			return false
		}

		// will reach here if response.Term <= node.currentTerm and response.Success == false
		// decrement nextIndex and retry the RPC, and keep repeating until it succeeds
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
			Client:       msg.Client,
		}

		return node.LeaderSendAE(replica_id, upper_index, client_obj, new_msg)

	} else {

		// AppendEntries for given client successful.
		node.nextIndex[replica_id] = upper_index + 1
		node.matchIndex[replica_id] = upper_index

		return true

	}

}

// Leader sending AppendEntries to all other replicas.
func (node *RaftNode) LeaderSendAEs(msg_type string, msg *protos.AppendEntriesMessage, upper_index int32, successful_write chan bool) {

	replica_id := int32(0)

	successes := int32(1)

	failures := int32(0)

	for _, client_obj := range node.peer_replica_clients {

		if replica_id == node.replica_id {
			replica_id++
			continue
		}

		go func(node *RaftNode, client_obj protos.ConsensusServiceClient, replica_id int32, upper_index int32, successful_write chan bool) {

			node.raft_node_mutex.Lock()
			//log.Printf("Lock on %d", replica_id)

			if node.LeaderSendAE(replica_id, upper_index, client_obj, msg) {

				tot_success := atomic.AddInt32(&successes, 1)

				if tot_success == (node.n_replicas)/2+1 { // write quorum achieved
					successful_write <- true // indicate to the calling function that the operation was perform successfully.
				}

			} else {
				tot_fail := atomic.AddInt32(&failures, 1)

				log.Printf("Sending AE FAILED for replica %v\n", replica_id)

				if tot_fail == (node.n_replicas+1)/2 {
					successful_write <- false // indicate to the calling function that the operation failed.
				}
			}
			//log.Printf("Trying to Unlock on %d\n", replica_id)
			node.raft_node_mutex.Unlock()
			//log.Printf("Unlock on %d", replica_id)

		}(node, client_obj, replica_id, upper_index, successful_write)

		replica_id++

	}

}

// HeartBeats is a goroutine that periodically makes leader
// send heartbeats as long as it is the leader
func (node *RaftNode) HeartBeats() {

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {

		<-ticker.C
		//log.Println("\nasking for Rlock")
		node.raft_node_mutex.RLock()
		//log.Println("\ngot Rlock")
		if node.state != Leader {

			node.raft_node_mutex.RUnlock()
			return
		}

		replica_id := node.replica_id

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

		node.raft_node_mutex.RUnlock()

		success := make(chan bool)
		node.LeaderSendAEs("HBEAT", hbeat_msg, int32(len(node.log)-1), success)
		<-success

	}
}
