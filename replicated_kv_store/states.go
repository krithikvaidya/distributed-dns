package main

import (
	"log"
	"context"

	"github.com/krithikvaidya/distributed-dns/replicated_kv_store/protos"
)

// ToFollower is called when you get a term higher than your own
func (node *RaftNode) ToFollower(ctx context.Context, term int32) {

	log.Printf("\nIn ToFollower, previous state: %v\n", node.state)
	prevState := node.state
	node.state = Follower
	node.currentTerm = term
	node.votedFor = -1

	node.persistToStorage()

	// CHECK: If node was a leader, start election timer. Else if it was a follower or
	// candidate, reset the election timer.

	if prevState == Leader {
		go node.RunElectionTimer(ctx)
	} else {
		go func() {
			node.electionResetEvent <- true
		}()
	}

	log.Printf("\nFinished ToFollower\n")
}

// ToCandidate is called when election timer runs out
// without heartbeat from leader
func (node *RaftNode) ToCandidate(ctx context.Context) {

	node.state = Candidate
	node.currentTerm++
	node.votedFor = node.node_meta.replica_id
	node.persistToStorage()
	//we can start an election for the candidate to become the leader
	node.StartElection(ctx)
}

// ToLeader is called when the candidate gets majority votes in election
func (node *RaftNode) ToLeader(ctx context.Context) {

	log.Printf("\nTransitioned to leader\n")
	// stop election timer since leader doesn't need it
	node.stopElectiontimer <- true

	node.state = Leader

	node.nextIndex = make([]int32, node.node_meta.n_replicas, node.node_meta.n_replicas)
	node.matchIndex = make([]int32, node.node_meta.n_replicas, node.node_meta.n_replicas)

	// initialize nextIndex, matchIndex
	for replica_id := int32(0); replica_id < node.node_meta.n_replicas; replica_id++ {

		if int32(replica_id) == node.node_meta.replica_id {
			continue
		}

		node.nextIndex[replica_id] = int32(len(node.log))
		node.matchIndex[replica_id] = int32(0)

	}

	// send no-op for synchronization
	// first obtain prevLogIndex and prevLogTerm

	var operation []string
	operation = append(operation, "NO-OP")

	node.log = append(node.log, protos.LogEntry{Term: node.currentTerm, Operation: operation, Clientid: " "})

	msg := &protos.AppendEntriesMessage{

		Term:         node.currentTerm,
		LeaderId:     node.node_meta.replica_id,
		LeaderCommit: node.commitIndex,
		LeaderAddr:   node.node_meta.nodeAddress,
		LatestClient: node.node_meta.latestClient,
	}

	node.persistToStorage()
	node.raft_node_mutex.Unlock()

	success := make(chan bool)
	node.LeaderSendAEs("NO-OP", msg, int32(len(node.log)-1), success)
	<-success

	node.raft_node_mutex.Lock()
	node.commitIndex++
	node.persistToStorage()
	node.raft_node_mutex.Unlock()
	node.commits_ready <- 1

	go node.HeartBeats(ctx)
}
