package main

import (
	"context"
	"log"

	"github.com/krithikvaidya/distributed-dns/replicated_kv_store/protos"
)

func (node *RaftNode) RequestVote(ctx context.Context, in *protos.RequestVoteMessage) (*protos.RequestVoteResponse, error) {

	// log.Printf("\nIn RequestVote. RWMutex write locked = %v\n", mutexasserts.RWMutexLocked(&node.raft_node_mutex))
	node.raft_node_mutex.Lock()

	latestLogIndex := int32(-1)
	latestLogTerm := int32(-1)

	if logLen := int32(len(node.log)); logLen > 0 {
		latestLogIndex = logLen - 1
		latestLogTerm = node.log[latestLogIndex].Term
	}

	log.Printf("\nReceived term: %v, My term: %v, My votedFor: %v\n", in.Term, node.currentTerm, node.votedFor)
	log.Printf("\nReceived latestLogIndex: %v, My latestLogIndex: %v, Received latestLogTerm: %v, My latestLogTerm: %v\n", in.LastLogIndex, latestLogIndex, in.LastLogTerm, latestLogTerm)

	// If the received message's term is greater than the replica's current term, transition to
	// follower (if not already a follower) and update term.
	if in.Term > node.currentTerm {
		node.ToFollower(in.Term)
	}

	// If ToFollower was called above, in.Term and node.currentTerm will be equal. If in.Term < node.currentTerm, reject vote.
	// If the candidate's log is not atleast as up-to-date as the replica's, reject vote.
	if (node.votedFor == in.CandidateId) ||
		((in.Term == node.currentTerm) && (node.votedFor == -1) &&
			(in.LastLogTerm > latestLogTerm || ((in.LastLogTerm == latestLogTerm) && (in.LastLogIndex >= latestLogIndex)))) {

		node.votedFor = in.CandidateId

		log.Printf("\nGranting vote\n")
		node.persistToStorage()
		node.raft_node_mutex.Unlock()
		return &protos.RequestVoteResponse{Term: node.currentTerm, VoteGranted: true}, nil

	} else {

		log.Printf("\nRejecting vote\n")
		node.raft_node_mutex.Unlock()
		return &protos.RequestVoteResponse{Term: node.currentTerm, VoteGranted: false}, nil

	}

}

func (node *RaftNode) AppendEntries(ctx context.Context, in *protos.AppendEntriesMessage) (*protos.AppendEntriesResponse, error) {

	log.Printf("\nreceived AE!\n")
	node.raft_node_mutex.Lock()
	log.Printf("\nlocked in AE\n")

	// term received is lesser than current term. CHECK: we don't reset election timer here.
	if node.currentTerm > in.Term {
		node.raft_node_mutex.Unlock()
		log.Printf("\nunlocked in AE\n")
		return &protos.AppendEntriesResponse{Term: node.currentTerm, Success: false}, nil

	} else if node.currentTerm < in.Term {

		// current term is lesser than received term, we transition into being a follower and reset timer and update term
		node.ToFollower(in.Term)

	} else if (node.currentTerm == in.Term) && (node.state == Candidate) {

		// the election for the current term has been won by another replica, and this replica should step down from candidacy
		node.ToFollower(in.Term)

	}

	// here we can be sure that the node's current term and the term in the message match, and that the node is not a leader or a
	// candidate.
	node.electionResetEvent <- true

	// we that the entry at PrevLogIndex (if it exists) has term PrevLogTerm
	if (in.PrevLogIndex == int32(-1)) || ((in.PrevLogIndex < int32(len(node.log))) && (node.log[in.PrevLogIndex].Term == in.PrevLogTerm)) {

		//log.Printf("\nin.PrevLogIndex : %d, in.PrevLogTerm : %d\n", in.PrevLogIndex, in.PrevLogTerm)
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
		flag := false

		for ; entryIndex < len(in.Entries); entryIndex++ {

			flag = true
			if logIndex == len(node.log) {

				// add new entry to log
				log.Printf("\nadd new entry to logs\n")
				node.log = append(node.log, *in.Entries[entryIndex])

			} else {

				// overwrite invalidated log entry
				log.Printf("\noverwrite invalidated log entry\n")
				node.log[logIndex] = *in.Entries[entryIndex]

			}

			logIndex++

		}

		if flag {
			node.persistToStorage()
		}

		//  If leaderCommit > commitIndex, set commitIndex = min(leaderCommit, index of last new entry)
		if in.LeaderCommit > node.commitIndex {

			old_commit_index := node.commitIndex

			log.Printf("\nin.LeaderCommit %v node.commitIndex %v int32(len(node.log)-1) %v\n", in.LeaderCommit, node.commitIndex, int32(len(node.log)-1))

			if in.LeaderCommit < int32(len(node.log)-1) {

				node.commitIndex = in.LeaderCommit

			} else {

				node.commitIndex = int32(len(node.log) - 1)

			}
			node.persistToStorage()

			log.Printf("\nunlocked in AE\n")
			node.raft_node_mutex.Unlock()

			node.commits_ready <- (node.commitIndex - old_commit_index)

		} else {
			log.Printf("\nunlocked in AE\n")
			node.raft_node_mutex.Unlock()
		}

		return &protos.AppendEntriesResponse{Term: node.currentTerm, Success: true}, nil // TODO: get node.currentTerm before unlock

	} else { //Reply false if log doesn’t contain an entry at prevLogIndex whose term matches prevLogTerm (§5.3)

		log.Printf("\nunlocked in AE\n")
		node.raft_node_mutex.Unlock()

		return &protos.AppendEntriesResponse{Term: node.currentTerm, Success: false}, nil // TODO: get node.currentTerm before unlock

	}

}
