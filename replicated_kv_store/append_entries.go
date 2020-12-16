package main

import (
	"context"

	"github.com/krithikvaidya/distributed-dns/replicated_kv_store/protos"
)

func (node *RaftNode) AppendEntries(ctx context.Context, in *protos.AppendEntriesMessage) (*protos.AppendEntriesResponse, error) {

	// ...
	return &protos.AppendEntriesResponse{}, nil

}
