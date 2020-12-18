package main

import (
	"math/rand"
	"time"
)

// ToFollower is called when you get a term higher than your own
func (node *RaftNode) ToFollower(term int) {
	node.node_state = Follower
	node.currentTerm = term
	node.votedFor = -1
	node.electionResetEvent = time.Now()

	go node.RunElectionTimer() // Once converted to follower
	//we need to run the election timeout in the background
}

// ToCandidate is called when election timer runs out
// without heartbeat from leader
func (node *RaftNode) ToCandidate() {
	node.node_state = Candidate
	node.currentTerm++
	node.electionResetEvent = time.Now()
	node.votedFor = node.replica_id

	node.StartElection()
	//we can start an election for the candidate to become the leader
}

// ToLeader is called when the candidate gets majority votes in election
func (node *RaftNode) ToLeader() {
	node.node_state = Leader

	go node.HeartBeats()
}

// RunElectionTimer runs an election of no heartbeat is received
func (node *RaftNode) RunElectionTimer() {
	duration := time.Duration(150+rand.Intn(150)) * time.Millisecond
	//150 - 300 ms random time was mentioned in the paper

	node.raft_node_mutex.Lock()
	start := node.currentTerm
	node.raft_node_mutex.Unlock()

	ticker := time.NewTicker(30 * time.Millisecond)
	defer ticker.Stop()

	for {
		<-ticker.C
		node.raft_node_mutex.Lock()
		if node.node_state != Candidate && node.node_state != Follower {
			node.raft_node_mutex.Unlock()
			return
		}

		if start != node.currentTerm {
			node.raft_node_mutex.Unlock()
			return
		}

		//If still follower and have not heard from leader nor voted for anyone else
		if elapsed := time.Since(node.electionResetEvent); elapsed >= duration {
			node.ToCandidate()
			node.raft_node_mutex.Unlock()
			return
		}
		node.raft_node_mutex.Unlock()
	}
}

//HeartBeats is a goroutine that periodically makes leader
//send heartbeats as long as it is the leader
func (node *RaftNode) HeartBeats() {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		//call function to send hearbeats to all other nodes
		<-ticker.C
		node.raft_node_mutex.Lock()
		if node.node_state != Leader {
			node.raft_node_mutex.Unlock()
			return
		}
		node.raft_node_mutex.Unlock()
	}
}

// StartElection is called when candidate is ready to start an election
func (node *RaftNode) StartElection() {
	//requestvote RPC
	//if election won call ToLeader
	//else call ToFollower
}
