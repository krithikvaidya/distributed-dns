package main

import (
	"log"

	"github.com/krithikvaidya/distributed-dns/replicated_kv_store/protos"
)

// ToFollower is called when you get a term higher than your own
func (node *RaftNode) ToFollower(term int32) {

	prevState := node.state
	node.state = Follower
	node.currentTerm = term
	node.votedFor = -1

	// If node was a leader or candidate, start election timer. Else if it was a candidate, reset the election timer.
	if prevState == Leader || prevState == Candidate {
		node.electionTimerRunning = true
		go node.RunElectionTimer()
	} else {
		if node.electionTimerRunning {
			node.electionResetEvent <- true
		}
	}

}

// ToCandidate is called when election timer runs out
// without heartbeat from leader
func (node *RaftNode) ToCandidate() {

	node.state = Candidate
	node.currentTerm++
	node.votedFor = node.replica_id

	//we can start an election for the candidate to become the leader
	node.StartElection()
}

// ToLeader is called when the candidate gets majority votes in election
func (node *RaftNode) ToLeader() {

	log.Printf("\nTransitioned to leader\n")
	// stop election timer since leader doesn't need it
	node.stopElectiontimer <- true
	node.electionTimerRunning = false

	node.state = Leader

	node.nextIndex = make([]int32, node.n_replicas, node.n_replicas)
	node.matchIndex = make([]int32, node.n_replicas, node.n_replicas)

	// initialize nextIndex, matchIndex
	for replica_id := int32(0); replica_id < node.n_replicas; replica_id++ {

		if int32(replica_id) == node.replica_id {
			continue
		}

		node.nextIndex[replica_id] = int32(len(node.log))
		node.matchIndex[replica_id] = int32(0)

	}

	// send no-op for synchronization
	// first obtain prevLogIndex and prevLogTerm

	prevLogIndex := int32(-1)
	prevLogTerm := int32(-1)

	if logLen := int32(len(node.log)); logLen > 0 {
		prevLogIndex = logLen - 1
		prevLogTerm = node.log[prevLogIndex].Term
	}

	var operation []string
	operation = append(operation, "NO-OP")

	node.log = append(node.log, protos.LogEntry{Term: node.currentTerm, Operation: operation})

	var entries []*protos.LogEntry
	entries = append(entries, &node.log[len(node.log)-1])

	msg := &protos.AppendEntriesMessage{

		Term:         node.currentTerm,
		LeaderId:     node.replica_id,
		PrevLogIndex: prevLogIndex,
		PrevLogTerm:  prevLogTerm,
		LeaderCommit: node.commitIndex,
		Entries:      entries,
	}

	node.raft_node_mutex.Unlock()

	success := make(chan bool)
	node.LeaderSendAEs("NO-OP", msg, int32(len(node.log)-1), success)
	<-success

	node.raft_node_mutex.Lock()
	node.commitIndex++
	node.raft_node_mutex.Unlock()
	node.commits_ready <- 1

	go node.HeartBeats()
}
