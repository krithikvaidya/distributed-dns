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
		// term received is lesser than current term
		if int32(node.currentTerm) > in.Term {
			return &protos.AppendEntriesResponse{Term: int32(node.currentTerm), Success: false}, nil
		} else if int32(node.currentTerm) <= in.Term { // currect term is lesser than received term, we transition into being a candidate and reset timer and update term
			node.currentTerm = int(in.Term)
			node.node_state = Candidate
			node.electionResetEvent = time.Time{}
		}
		// we start from prevlogindex, check if stored values are equal to that in the leader if no, then it updates value and goes back doing the same till they are equal
		i := in.PrevLogIndex
		for ; int32(node.log[i]) != in.Entries[i] && i >= 0; i-- {
			node.log[i] = int(in.Entries[i])
		}
		// if all entries arent in the node then it adds those values
		for i = in.PrevLogIndex + 1; i < int32(len(in.Entries)); i++ {
			node.log[i] = int(in.Entries[i])
		}
		//  If leaderCommit > commitIndex, set commitIndex = min(leaderCommit, index of last new entry)
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
