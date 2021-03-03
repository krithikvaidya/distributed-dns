package main

import (
	"context"
	"log"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/krithikvaidya/distributed-dns/replicated_kv_store/protos"
)

// RunElectionTimer runs an election and initiates transition to candidate
// if a heartbeat/appendentries RPC is not received within the timeout duration.
func (node *RaftNode) RunElectionTimer(parent_ctx context.Context) {

	// 150 - 300 ms random timeout was mentioned in the paper
	duration := time.Duration(500+rand.Intn(200)) * time.Millisecond

	/*
	 * Make sure that election is not run when context is cancelled by prioritizing
	 * it in the `select` statement.
	 */
	select {

	case <-time.After(duration): // for timeout to call election

		node.raft_node_mutex.Lock()

		// by the time the lock was acquired, if either
		// 1. context cancel has occured
		// 2. the electionResetEvent channel has been written to
		// 3. the stopElectionTimer channel has been written to
		// don't transition to candidate.

		select {

		// prioritize checking if context is cancelled.
		case <-parent_ctx.Done():
			return

		default:

			select {

			case <-node.stopElectiontimer: // to stop timer
				defer node.raft_node_mutex.Unlock()
				return

			case <-node.electionResetEvent: // to reset timer when heartbeat/msg received

				node.raft_node_mutex.Unlock()
				go node.RunElectionTimer(parent_ctx)
				return

			default:
				break // break out of select block

			}

		}

		log.Printf("\nElection timer runs out.\n")

		// if node was a follower, transition to candidate and start election
		// if node was already candidate, restart election

		node.ToCandidate(parent_ctx)

		node.raft_node_mutex.Unlock()
		return

	case <-node.stopElectiontimer: //to stop timer
		return

	case <-node.electionResetEvent: //to reset timer when heartbeat/msg received
		go node.RunElectionTimer(parent_ctx)
		return

	}
}

// StartElection is called when candidate is ready to start an election
func (node *RaftNode) StartElection(ctx context.Context) {

	var received_votes int32 = 1
	replica_id := int32(0)

	for _, client_obj := range node.meta.peer_replica_clients {

		if replica_id == node.meta.replica_id {
			replica_id++
			continue
		}

		go func(ctx context.Context, node *RaftNode, client_obj protos.ConsensusServiceClient, replica_id int32) {

			node.raft_node_mutex.RLock()
			// log.Printf("\nRLock in StartElection\n")

			latestLogIndex := int32(-1)
			latestLogTerm := int32(-1)

			if logLen := int32(len(node.log)); logLen > 0 {
				latestLogIndex = logLen - 1
				latestLogTerm = node.log[latestLogIndex].Term
			}

			args := protos.RequestVoteMessage{
				Term:         node.currentTerm,
				CandidateId:  node.meta.replica_id,
				LastLogIndex: latestLogIndex,
				LastLogTerm:  latestLogTerm,
			}

			node.raft_node_mutex.RUnlock()

			//request vote and get reply
			response, err := client_obj.RequestVote(ctx, &args)

			node.raft_node_mutex.Lock()

			if err == nil {

				// by the time the RPC call returns an answer, this replica might have already transitioned to another state.

				if node.state != Candidate {
					node.raft_node_mutex.Unlock()
					return
				}

				if response.Term > node.currentTerm { // the response node has higher term than current one

					node.ToFollower(ctx, response.Term)
					node.raft_node_mutex.Unlock()
					return

				} else if response.Term == node.currentTerm {

					if response.VoteGranted {

						// log.Printf("\nReceived vote from %v\n", replica_id)
						votes := int(atomic.AddInt32(&received_votes, 1))

						if votes*2 > int(node.meta.n_replicas) { // won the Election
							node.ToLeader(ctx)

							return
						}

					}

				}

			} else {

				// log.Printf("\nError in requesting vote from replica %v: %v", replica_id, err.Error())

			}

			node.raft_node_mutex.Unlock()

		}(ctx, node, client_obj, replica_id)

		replica_id++

	}

	go node.RunElectionTimer(ctx) // begin the timer during which this candidate waits for votes
}
