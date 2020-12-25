package main

import (
	"context"
	"log"

	"github.com/krithikvaidya/distributed-dns/replicated_kv_store/protos"
)

func (node *RaftNode) RequestVote(ctx context.Context, in *protos.RequestVoteMessage) (*protos.RequestVoteResponse, error) {

	// fmt.Printf("\nreceived requestvote\n")

	// log.Printf("\nIn RequestVote. rw write locked = %v\n", mutexasserts.RWMutexLocked(&node.raft_node_mutex))
	node.raft_node_mutex.Lock()

	node_current_term := node.currentTerm
	latestLogIndex := int32(-1)
	latestLogTerm := int32(-1)

	if logLen := int32(len(node.log)); logLen > 0 {
		latestLogIndex = logLen - 1
		latestLogTerm = node.log[latestLogIndex].Term
	}

	log.Printf("\nReceived term: %v, My term: %v, My votedFor: %v\n", in.Term, node_current_term, node.votedFor)
	log.Printf("\nReceived latestLogIndex: %v, My latestLogIndex: %v, Received latestLogTerm: %v, My latestLogTerm: %v\n", in.LastLogIndex, latestLogIndex, in.LastLogTerm, latestLogTerm)

	// If the received message's term is greater than the replica's current term, transition to
	// follower (if not already a follower) and update term.
	if in.Term > node_current_term {
		node.ToFollower(in.Term)
		log.Printf("\nAfter tofollower, my term %v\n", node.currentTerm)
	}

	// Grant vote if the received message's term is not lesser than the replica's term, and if the
	// candidate's log is atleast as up-to-date as the replica's.
	if (node.votedFor == in.CandidateId) ||
		((in.Term > node_current_term) && (node.votedFor == -1) &&
			(in.LastLogTerm > latestLogTerm || ((in.LastLogTerm == latestLogTerm) && (in.LastLogIndex >= latestLogIndex)))) {

		if node.electionTimerRunning {
			node.electionResetEvent <- true
		}
		node.votedFor = in.CandidateId

		log.Printf("\nGranting vote\n")

		node.raft_node_mutex.Unlock()
		return &protos.RequestVoteResponse{Term: in.Term, VoteGranted: true}, nil

	} else {

		log.Printf("\nRejecting vote\n")
		node.raft_node_mutex.Unlock()
		return &protos.RequestVoteResponse{Term: node_current_term, VoteGranted: false}, nil

	}

}
