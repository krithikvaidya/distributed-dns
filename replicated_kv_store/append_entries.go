package main

import (
	"context"
	"time"

	"github.com/krithikvaidya/distributed-dns/replicated_kv_store/protos"
)

func (node *RaftNode) AppendEntries(ctx context.Context, in *protos.AppendEntriesMessage) (*protos.AppendEntriesResponse, error) {
	select {
	case <-ctx.Done():
		return &protos.AppendEntriesResponse{Term: int32(node.currentTerm), Success: false}, nil
	default:
		if int32(node.currentTerm) > in.Term {
			return &protos.AppendEntriesResponse{Term: int32(node.currentTerm), Success: false}, nil
		} else if node.log[in.PrevLogIndex] != int(in.PrevLogTerm) {
			return &protos.AppendEntriesResponse{Term: int32(node.currentTerm), Success: false}, nil
		} else if int32(node.currentTerm) < in.Term {
			node.currentTerm = int(in.Term)
			node.node_state = Candidate
			node.electionResetEvent = time.Time{}
		}
		i := in.PrevLogIndex
		for ; int32(node.log[i]) != in.Entries[i]; i-- {
			node.log[i] = int(in.Entries[i])
		}
		for i = in.PrevLogIndex + 1; i < int32(len(in.Entries)); i++ {
			node.log[i] = int(in.Entries[i])
		}
		if len(in.Entries) > node.commitIndex {
			if int32(len(in.Entries)) < in.PrevLogIndex+1 {
				node.commitIndex = len(in.Entries)
			} else {
				node.commitIndex = int(in.PrevLogIndex)
			}
		}
	}
	// ...
	return &protos.AppendEntriesResponse{Term: int32(node.currentTerm), Success: true}, nil

}
