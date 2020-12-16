package main

import (
	"context"

	"github.com/krithikvaidya/distributed-dns/replicated_kv_store/protos"
)

func (node *RaftNode) RequestVote(ctx context.Context, in *protos.RequestVoteMessage) (*protos.RequestVoteResponse, error) {

	// ...
	return &protos.RequestVoteResponse{}, nil

}
