package main

import (
	"context"
	"log"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/krithikvaidya/distributed-dns/replicated_kv_store/protos"
)

// RPC declared in protos/replica.proto.
// When a replica performs the gRPC dial to another replica and obtains the
// corresponding client stub, it will invoke this RPC to inform the other replica
// that it has connected. (This RPC might be unnecessary if the initial setup procedure is refined)
func (node *RaftNode) ReplicaReady(ctx context.Context, in *empty.Empty) (*empty.Empty, error) {

	log.Printf("\nReceived ReplicaReady Notification\n")

	// log.Printf("\nRWMutex write locked = %v\n", mutexasserts.RWMutexLocked(&node.raft_node_mutex))
	node.raft_node_mutex.Lock()

	node.replicas_ready += 1

	if node.replicas_ready == node.n_replicas-1 {

		// Using defer does not work here. Not sure why
		go func(node *RaftNode) {

			node.ready_chan <- true

		}(node)

		log.Printf("\nAll replicas have connected.\n")

	}

	node.raft_node_mutex.Unlock()
	// log.Printf("\nRWMutex write locked = %v\n", mutexasserts.RWMutexLocked(&node.raft_node_mutex))

	return &empty.Empty{}, nil
}

func (node *RaftNode) RequestVote(ctx context.Context, in *protos.RequestVoteMessage) (*protos.RequestVoteResponse, error) {

	// log.Printf("\nIn RequestVote. RWMutex write locked = %v\n", mutexasserts.RWMutexLocked(&node.raft_node_mutex))
	node.raft_node_mutex.Lock()

	node_current_term := node.currentTerm
	latestLogIndex := int32(-1)
	latestLogTerm := int32(-1)

	if logLen := int32(len(node.log)); logLen > 0 {
		latestLogIndex = logLen - 1
		latestLogTerm = node.log[latestLogIndex].Term
	}

	log.Printf("\nReceived term: %v, My term: %v, My votedFor: %v\n", in.Term, node_current_term, node.votedFor)
	log.Printf("\nReceived latestLogIndex: %v, My latestLogIndex: %v, Received latestLogTerm: %v, My latestLogTerm: %v\n", in.LastLogIndex, latestLogIndex, in.LastLogTerm, latestLogTerm)

	// If the received message's term is greater than the replica's current term, transition to
	// follower (if not already a follower) and update term.
	if in.Term > node_current_term {
		node.ToFollower(in.Term)
		// log.Printf("\nAfter tofollower, my term %v\n", node.currentTerm)
	}

	// Grant vote if the received message's term is not lesser than the replica's term, and if the
	// candidate's log is atleast as up-to-date as the replica's.
	if (node.votedFor == in.CandidateId) ||
		((in.Term > node_current_term) && (node.votedFor == -1) &&
			(in.LastLogTerm > latestLogTerm || ((in.LastLogTerm == latestLogTerm) && (in.LastLogIndex >= latestLogIndex)))) {

		if node.electionTimerRunning {
			node.electionResetEvent <- true
		}
		node.votedFor = in.CandidateId

		log.Printf("\nGranting vote\n")
		node.persistToStorage()
		node.raft_node_mutex.Unlock()
		return &protos.RequestVoteResponse{Term: in.Term, VoteGranted: true}, nil

	} else {

		log.Printf("\nRejecting vote\n")
		node.persistToStorage()
		node.raft_node_mutex.Unlock()
		return &protos.RequestVoteResponse{Term: node_current_term, VoteGranted: false}, nil

	}

}

func (node *RaftNode) AppendEntries(ctx context.Context, in *protos.AppendEntriesMessage) (*protos.AppendEntriesResponse, error) {

	node.raft_node_mutex.Lock()

	// term received is lesser than current term
	if node.currentTerm > in.Term {
		node.raft_node_mutex.Unlock()
		return &protos.AppendEntriesResponse{Term: node.currentTerm, Success: false}, nil

	} else if node.currentTerm < in.Term {

		// current term is lesser than received term, we transition into being a follower and reset timer and update term
		node.ToFollower(in.Term)

	} else if (node.currentTerm == in.Term) && (node.state == Candidate) {

		// the election for the current term has been won by another replica, and this replica should step down from candidacy
		node.ToFollower(in.Term)

	}

	// here we can be sure that the node's current term and the term in the message match.
	if node.electionTimerRunning {
		node.electionResetEvent <- true
	}

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
				log.Printf("\nadd new entry to logs\n")
				node.log = append(node.log, *in.Entries[entryIndex])

			} else {

				// overwrite invalidated log entry
				log.Printf("\noverwrite invalidated log entry\n")
				node.log[logIndex] = *in.Entries[entryIndex]

			}

			logIndex++

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

			node.raft_node_mutex.Unlock()

			node.commits_ready <- (node.commitIndex - old_commit_index)

		} else {
			node.persistToStorage()
			node.raft_node_mutex.Unlock()
		}

		return &protos.AppendEntriesResponse{Term: node.currentTerm, Success: true}, nil

	} else { //Reply false if log doesn’t contain an entry at prevLogIndex whose term matches prevLogTerm (§5.3)

		node.persistToStorage()
		node.raft_node_mutex.Unlock()

		return &protos.AppendEntriesResponse{Term: node.currentTerm, Success: false}, nil

	}

}
