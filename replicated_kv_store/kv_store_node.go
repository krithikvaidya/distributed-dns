package main

import (
	"sync"
	"time"
)

const (
	Follower int = iota
	Candidate
	Leader
	Down
)

type RaftNode struct {
	replica_id       int
	peer_replica_ids []int
	raft_node_mutex  sync.Mutex

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
	nextIndex  map[int]int
	matchIndex map[int]int
}

func GetState(state int) string {

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
