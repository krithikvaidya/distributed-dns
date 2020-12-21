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
		} else if int32(node.currentTerm) < in.Term { // currect term is lesser than received term, we transition into being a candidate and reset timer and update term
			node.currentTerm = in.Term
			node.state = Follower
			node.electionResetEvent = time.Time{}
		}

		if in.PrevLogIndex == -1 || node.log[in.PrevLogIndex].term == in.PrevLogTerm && in.PrevLogIndex {
			logIndex := in.PrevLogIndex + 1
			entryIndex := 0
			for entryIndex < len(in.Entries) {
				// we start from prevlogindex, check if stored values are equal to that in the leader if no, then it updates value and goes ahead doing the same
				if node.log[logIndex].value != in.Entries[entryIndex].value || node.log[logIndex].term != in.Entries[entryIndex].term {
					node.log[logIndex].value = in.Entries[entryIndex].value
					node.log[logIndex].term = in.Entries[entryIndex].term
					node.log[logIndex].operation = in.Entries[entryIndex].operation
				}
				logIndex++
				entryIndex++
			}

			//  If leaderCommit > commitIndex, set commitIndex = min(leaderCommit, index of last new entry)
			if in.leaderCommit > node.commitIndex {
				if in.leaderCommit < in.PrevLogIndex+1 {
					node.commitIndex = in.leaderCommit
				} else {
					node.commitIndex = in.PrevLogIndex
				}
			}
			return &protos.AppendEntriesResponse{Term: int32(node.currentTerm), Success: true}, nil
		} else { //Reply false if log doesn’t contain an entry at prevLogIndex whose term matches prevLogTerm (§5.3)
			return &protos.AppendEntriesResponse{Term: int32(node.currentTerm), Success: false}, nil
		}

	}
	// ...
}
