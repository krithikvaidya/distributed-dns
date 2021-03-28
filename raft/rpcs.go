package raft

import (
	"context"
	"log"

	"github.com/krithikvaidya/distributed-dns/raft/protos"
)

// Implements the functionality involved when a replica recieves a RequestVote RPC.
// If the received message's term is greater than the replica's current term, transition to
// follower (if not already a follower) and update term. If in.Term < node.currentTerm, reject vote.
// If the candidate's log is not atleast as up-to-date as the replica's, reject vote.
func (node *RaftNode) RequestVote(ctx context.Context, in *protos.RequestVoteMessage) (*protos.RequestVoteResponse, error) {

	log.Printf("\nReceived RequestVote RPC\n")

	node.GetLock("RequestVote")

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
		node.ToFollower(node.Meta.Master_ctx, in.Term)
	}

	// If ToFollower was called above, in.Term and node.currentTerm will be equal. If in.Term < node.currentTerm, reject vote.
	// If the candidate's log is not atleast as up-to-date as the replica's, reject vote.
	if (node.votedFor == in.CandidateId) ||
		((in.Term == node.currentTerm) && (node.votedFor == -1) &&
			(in.LastLogTerm > latestLogTerm || ((in.LastLogTerm == latestLogTerm) && (in.LastLogIndex >= latestLogIndex)))) {

		node.votedFor = in.CandidateId

		log.Printf("\nGranting vote to %v\n", in.CandidateId)
		node.PersistToStorage()
		node.ReleaseLock("RequestVote1")
		return &protos.RequestVoteResponse{Term: in.Term, VoteGranted: true}, nil

	} else {

		log.Printf("\nRejecting vote to %v\n", in.CandidateId)
		node.ReleaseLock("RequestVote2")
		return &protos.RequestVoteResponse{Term: in.Term, VoteGranted: false}, nil

	}

}

// Implements the functionality involved when a replica recieves an AppendEntries RPC, as the
// Raft paper describes it.
func (node *RaftNode) AppendEntries(ctx context.Context, in *protos.AppendEntriesMessage) (*protos.AppendEntriesResponse, error) {

	node.GetLock("AppendEntries1")

	// if the current replica's term is greater than the term in the received message
	if node.currentTerm > in.Term {

		// Check if the logs were replicated earlier, i.e. if entry at PrevLogIndex (if it exists)
		// has term PrevLogTerm. If yes, check if the logs match.
		if (in.PrevLogIndex == int32(-1)) || ((in.PrevLogIndex < int32(len(node.log))) && (node.log[in.PrevLogIndex].Term == in.PrevLogTerm)) {

			entryIndex := 0

			for logIndex := int(in.PrevLogIndex + 1); (entryIndex < len(in.Entries)) && (logIndex < len(node.log)); logIndex++ {

				// we start from prevlogindex and try to find the first mismatch, if any
				if node.log[logIndex].Term != in.Entries[entryIndex].Term {
					break
				}

				entryIndex++

			}

			// if entryIndex has reached the end of the entries received in the message, then all the logs received in the
			// message have been replicated.
			if entryIndex == len(in.Entries) {
				node.ReleaseLock("AppendEntries1")
				return &protos.AppendEntriesResponse{Term: node.currentTerm, Success: true}, nil
			} else {
				node.ReleaseLock("AppendEntries2")
				log.Printf("\nResponding False in AE because entryIndex != len(in.Entries)")
				return &protos.AppendEntriesResponse{Term: node.currentTerm, Success: false}, nil
			}

		} else {
			// entry at PrevLogIndex does not have term PrevLogTerm
			node.ReleaseLock("AppendEntries3")
			log.Printf("\nResponding False in AE because entry at PrevLogIndex does not have term PrevLogTerm")
			return &protos.AppendEntriesResponse{Term: node.currentTerm, Success: false}, nil
		}

	} else if node.currentTerm < in.Term {

		// current term is lesser than received term, we transition into being a follower and reset timer and update term
		node.ToFollower(node.Meta.Master_ctx, in.Term)
	} else if (node.currentTerm == in.Term) && (node.state == Candidate) {

		// the election for the current term has been won by another replica, and this replica should step down from candidacy
		node.ToFollower(node.Meta.Master_ctx, in.Term)

	}

	// here we can be sure that the node's current term and the term in the message match, and that the current replica
	// is not a leader or a candidate.
	node.electionResetEvent <- true

	node.Meta.leaderAddress = in.LeaderAddr // gets the leaders address

	// we ensure that the entry at PrevLogIndex (if it exists) has term PrevLogTerm
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
				log.Printf("\nAdd new entry to logs\n")
				node.log = append(node.log, *in.Entries[entryIndex])

			} else {

				// overwrite invalidated log entry
				log.Printf("\nOverwrite invalidated log entry\n")
				node.log[logIndex] = *in.Entries[entryIndex]

			}

			logIndex++

		}

		if in.LeaderCommit > node.commitIndex {

			old_commit_index := node.commitIndex

			node.Meta.latestClient = in.LatestClient // stores the id of the most recent client

			for i := node.commitIndex + 1; i <= in.LeaderCommit && i < int32(len(node.log)); i++ {

				node.trackMessage[node.log[i].Clientid] = node.log[i].Operation //Updates the trackMessages for each client to the latest operation

			}

			if in.LeaderCommit < int32(len(node.log)) {

				node.commitIndex = in.LeaderCommit

			} else {

				node.commitIndex = int32(len(node.log) - 1)

			}

			node.commits_ready <- (node.commitIndex - old_commit_index)

		}

		node.PersistToStorage()
		node.ReleaseLock("AppendEntries4")

		return &protos.AppendEntriesResponse{Term: in.Term, Success: true}, nil

	} else { //Reply false if log doesn’t contain an entry at prevLogIndex whose term matches prevLogTerm (§5.3)

		node.ReleaseLock("AppendEntries5")
		log.Printf("\nResponding False in AE because log doesn’t contain an entry at prevLogIndex whose term matches prevLogTerm 2")
		return &protos.AppendEntriesResponse{Term: in.Term, Success: false}, nil

	}

}
