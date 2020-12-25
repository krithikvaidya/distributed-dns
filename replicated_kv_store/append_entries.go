package main

import (
	"context"

	"github.com/krithikvaidya/distributed-dns/replicated_kv_store/protos"
)

func (node *RaftNode) AppendEntries(ctx context.Context, in *protos.AppendEntriesMessage) (*protos.AppendEntriesResponse, error) {

	node.raft_node_mutex.Lock()

	// term received is lesser than current term
	if node.currentTerm > in.Term {

		node.raft_node_mutex.Unlock()
		return &protos.AppendEntriesResponse{Term: node.currentTerm, Success: false}, nil

	} else if node.currentTerm < in.Term {

		// current term is lesser than received term, we transition into being a candidate and reset timer and update term
		node.ToFollower(in.Term)

	} else if (node.currentTerm == in.Term) && (node.state == Candidate) {

		// the election for the current term has been won by another replica, and this replica should step down from candidacy
		node.ToFollower(in.Term)

	}

	// here we can be sure that the node's current term and the term in the message match.

	// we that the entry at PrevLogIndex (if it exists) has term PrevLogTerm
	if (in.PrevLogIndex == int32(-1)) || ((in.PrevLogIndex < int32(len(node.log))) && (node.log[in.PrevLogIndex].Term == in.PrevLogTerm)) {

		logIndex := int(in.PrevLogIndex + 1)
		entryIndex := 0

		for ; (entryIndex < len(in.Entries)) && (logIndex < len(node.log)); logIndex++ {

			// we start from prevlogindex and try to find the first mismatch, if any
			if node.log[logIndex].Term != in.Entries[entryIndex].Term {
				break
			}

			entryIndex++

		}

		// at this point, logIndex has either reached the end of the log (or the first conflicting entry), and/or entryIndex has reached the end
		// of the message's entries. if entryIndex has reached the end, it means that there is nothing new to add to the candidate's log.
		for ; entryIndex < len(in.Entries); entryIndex++ {

			if logIndex == len(node.log) {

				// add new entry to log
				node.log = append(node.log, *in.Entries[entryIndex])

			} else {

				// overwrite invalidated log entry
				node.log[logIndex] = *in.Entries[entryIndex]

			}

			logIndex++

		}

		//  If leaderCommit > commitIndex, set commitIndex = min(leaderCommit, index of last new entry)
		if in.LeaderCommit > node.commitIndex {

			if in.LeaderCommit < int32(len(node.log)-1) {

				node.commitIndex = in.LeaderCommit

			} else {

				node.commitIndex = int32(len(node.log) - 1)

			}
		}

		node.raft_node_mutex.Unlock()
		return &protos.AppendEntriesResponse{Term: node.currentTerm, Success: true}, nil

	} else { //Reply false if log doesn’t contain an entry at prevLogIndex whose term matches prevLogTerm (§5.3)

		node.raft_node_mutex.Unlock()
		return &protos.AppendEntriesResponse{Term: node.currentTerm, Success: false}, nil

	}

}
