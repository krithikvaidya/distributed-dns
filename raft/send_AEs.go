package raft

import (
	"context"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/krithikvaidya/distributed-dns/raft/protos"
	"google.golang.org/grpc"
)

// To send AppendEntry to single replica, and retry if needed (called by LeaderSendAEs defined below).
func (node *RaftNode) LeaderSendAE(parent_ctx context.Context, replica_id int32, upper_index int32, client_obj protos.ConsensusServiceClient, msg *protos.AppendEntriesMessage) (status bool) {

	var response *protos.AppendEntriesResponse
	var err error

	// Call the AppendEntries RPC for the given client
	ctx, _ := context.WithTimeout(parent_ctx, 40*time.Millisecond)
	response, err = client_obj.AppendEntries(ctx, msg)

	if err != nil {

		// If there was a problem connecting to the gRPC server and invoking the RPC,
		// we retry dialing to that server and performing the RPC. If this fails too, return false.
		connxn, err := grpc.Dial(":500"+strconv.Itoa(int(replica_id)), grpc.WithInsecure())

		// Obtain client stub
		cli := protos.NewConsensusServiceClient(connxn)

		node.Meta.peer_replica_clients[replica_id] = cli

		// Call the AppendEntries RPC for the given client
		ctx, _ := context.WithTimeout(parent_ctx, 40*time.Millisecond)
		response, err = client_obj.AppendEntries(ctx, msg)

		if err != nil {
			return false
		}
	}

	node.GetLock("LeaderSendAE")

	if response.Success == false {

		if node.state != Leader {
			node.ReleaseLock("LeaderSendAE")
			return false
		}

		if response.Term > node.currentTerm {

			node.ToFollower(parent_ctx, response.Term)
			node.ReleaseLock("LeaderSendAE")
			return false
		}

		// will reach here if response.Term == node.currentTerm and response.Success == false
		// Keep decrementing nextIndex and retrying the RPC until it succeeds
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
			LeaderId:     node.Meta.replica_id,
			PrevLogIndex: prevLogIndex,
			PrevLogTerm:  prevLogTerm,
			LeaderCommit: node.commitIndex,
			Entries:      entries,
			LeaderAddr:   node.Meta.nodeAddress,
			LatestClient: node.Meta.latestClient,
		}

		node.ReleaseLock("LeaderSendAE")
		return node.LeaderSendAE(parent_ctx, replica_id, upper_index, client_obj, new_msg)

	} else { //response.Success == true

		if node.currentTerm < response.Term {

			node.ToFollower(parent_ctx, response.Term)

		} else {

			node.nextIndex[replica_id] = upper_index + 1
			node.matchIndex[replica_id] = upper_index

		}

		node.ReleaseLock("LeaderSendAE")
		return true

	}

}

// Called when the replica wants to send AppendEntries to all other replicas.
func (node *RaftNode) LeaderSendAEs(msg_type string, msg *protos.AppendEntriesMessage, upper_index int32, successful_write chan bool) {

	replica_id := int32(0)

	successes := int32(1)
	failures := int32(0)

	for _, client_obj := range node.Meta.peer_replica_clients {

		if replica_id == node.Meta.replica_id {
			replica_id++
			continue
		}

		if client_obj == nil {
			replica_id++
			continue
		}

		go func(node *RaftNode, client_obj protos.ConsensusServiceClient, replica_id int32, upper_index int32, successful_write chan bool) {

			node.GetRLock("LeaderSendAEs1")

			prevLogIndex := node.nextIndex[replica_id] - 1
			prevLogTerm := int32(-1)

			if prevLogIndex >= 0 {
				prevLogTerm = node.log[prevLogIndex].Term
			}

			msg.PrevLogIndex = prevLogIndex
			msg.PrevLogTerm = prevLogTerm

			var entries []*protos.LogEntry

			for i := int32(prevLogIndex + 1); i <= upper_index; i++ {
				entries = append(entries, &node.log[i])
			}

			node.ReleaseRLock("LeaderSendAEs")

			msg.Entries = entries

			ctx, _ := context.WithTimeout(context.Background(), 20*time.Millisecond)

			if node.LeaderSendAE(ctx, replica_id, upper_index, client_obj, msg) {

				tot_success := atomic.AddInt32(&successes, 1)

				if tot_success == (node.Meta.n_replicas)/2+1 { // write quorum achieved
					successful_write <- true // indicate to the calling function that the operation was performed successfully.
				}

			} else {
				tot_fail := atomic.AddInt32(&failures, 1)

				if tot_fail == (node.Meta.n_replicas+1)/2 {
					successful_write <- false // indicate to the calling function that the operation failed.
				}
			}

		}(node, client_obj, replica_id, upper_index, successful_write)

		replica_id++

	}

}

// HeartBeats is a goroutine that periodically sends heartbeats as long as the replicas thinks it's a leader
func (node *RaftNode) HeartBeats(ctx context.Context) {

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	/*
	 * The following select statements are to make sure that the context being
	 * cancelled is prioritized more than the ticker. This makes sure that the
	 * heartbeat procedure can be cancelled at any time.
	 */
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			select {
			case <-ctx.Done():
				return
			default:
			}

			node.GetRLock("HeartBeats")

			if node.state != Leader {

				node.ReleaseRLock("HeartBeats1")
				return
			}

			node.ReleaseRLock("HeartBeats2")

			hbeat_msg := &protos.AppendEntriesMessage{

				Term:         node.currentTerm,
				LeaderId:     node.Meta.replica_id,
				LeaderCommit: node.commitIndex,
				LeaderAddr:   node.Meta.nodeAddress,
				LatestClient: node.Meta.latestClient,
			}

			success := make(chan bool)
			node.LeaderSendAEs("HBEAT", hbeat_msg, int32(len(node.log)-1), success)
			<-success
		}
	}
}
