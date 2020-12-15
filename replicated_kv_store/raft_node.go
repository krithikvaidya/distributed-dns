package main

import (
	"sync"
	"time"
)

type RaftNodeState int

const (
	Follower RaftNodeState = iota
	Candidate
	Leader
	Down
)

type RaftNode struct {
	replica_id       int
	peer_replica_addresses []string
	raft_node_mutex  sync.Mutex
	node_state       RaftNodeState

	// States mentioned in figure 2 of the paper:

	// State to be maintained on all replicas (TODO: persist)
	currentTerm int
	votedFor    int
	log         []LogEntry

	// State to be maintained on all replicas
	commitIndex        int
	lastApplied        int
	state              CMState
	electionResetEvent time.Time

	// State to be maintained on the leader
	nextIndex  []int
	matchIndex []int
}

func GetState(state RaftNodeState) string {

	switch state {

	case Follower:
		return "Follower"
	case Candidate:
		return "Candidate"
	case Leader:
		return "Leader"
	case Down:
		return "Down"

	}

}

func InitializeNode (rid int, praddrs []int) {

	rn := &RaftNode {

		replica_id: rid,
		peer_replica_addresses: praddrs,
		node_state: Follower,
		
		currentTerm: 0,  // unpersisted
		votedFor: -1,
		log: make([]int, 10000)

		commitIndex: 0,
		lastApplied: 0,

	}

	return rn

}