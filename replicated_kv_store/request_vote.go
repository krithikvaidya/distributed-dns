package main

import (
	"context"
	"time"

	"github.com/krithikvaidya/distributed-dns/replicated_kv_store/protos"
)

func (node *RaftNode) RequestVote(ctx context.Context, in *protos.RequestVoteMessage) (*protos.RequestVoteResponse, error) {
	node.raft_node_mutex.Lock()
	node_current_term := node.currentTerm
	node.raft_node_mutex.Unlock()
	select {
	case <-ctx.Done():
		return &protos.RequestVoteResponse{Term: int32(node_current_term), VoteGranted: false}, nil
	default:
		if in.Term > int32(node_current_term) {
			node.ToFollower(int(in.Term))
		} else if (in.Term == int32(node.currentTerm) && node.votedFor == -1) || (node.replica_id == int(in.CandidateId) && node.log[len(node.log)-1] == int(in.LastLogIndex)) {
			node.raft_node_mutex.Lock()
			node.electionResetEvent = time.Time{}
			node.raft_node_mutex.Unlock()
			return &protos.RequestVoteResponse{Term: in.Term, VoteGranted: true}, nil
		} else {
			return &protos.RequestVoteResponse{Term: int32(node_current_term), VoteGranted: false}, nil
		}
	}
	return &protos.RequestVoteResponse{Term: int32(node_current_term), VoteGranted: false}, nil
}
