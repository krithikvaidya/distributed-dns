package main

import (
	"context"

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

	// If the received message's term is greater than the replica's current term, transition to
	// follower (if not already a follower) and update term.
	if in.Term > node_current_term {
		node.ToFollower(in.Term)
	}

	// Grant vote if the received message's term is not lesser than the replica's term, and if the
	// candidate's log is atleast as up-to-date as the replica's.
	if (node.votedFor == in.CandidateId) ||
		((in.Term > node.currentTerm) && (node.votedFor == -1) &&
			(in.LastLogTerm > latestLogTerm || ((in.LastLogTerm == latestLogTerm) && (in.LastLogIndex >= latestLogIndex)))) {

		node.electionResetEvent <- true
		node.votedFor = in.CandidateId

		node.raft_node_mutex.Unlock()
		return &protos.RequestVoteResponse{Term: in.Term, VoteGranted: true}, nil

	} else {

		node.raft_node_mutex.Unlock()
		return &protos.RequestVoteResponse{Term: node_current_term, VoteGranted: false}, nil

	}

}
