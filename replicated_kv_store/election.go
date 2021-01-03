package main

import (
	"context"
	"log"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/krithikvaidya/distributed-dns/replicated_kv_store/protos"
)

// RunElectionTimer runs an election if no heartbeat is received
func (node *RaftNode) RunElectionTimer() {
	duration := time.Duration(150+rand.Intn(150)) * time.Millisecond
	//150 - 300 ms random time was mentioned in the paper

	// go node.ElectionStopper(start)

	select {

	case <-time.After(duration): //for timeout to call election

		log.Printf("\nElection timer runs out.\n")
		// if node was a follower, transition to candidate and start election
		// if node was already candidate, restart election
		node.raft_node_mutex.Lock()
		// log.Printf("\nLocked in RunElectionTimer\n")

		node.electionTimerRunning = false
		node.ToCandidate()

		// log.Printf("\nUnlocked in AppendEntries\n")

		node.raft_node_mutex.Unlock()
		return

	case <-node.stopElectiontimer: //to stop timer
		node.electionTimerRunning = false
		return

	case <-node.electionResetEvent: //to reset timer when heartbeat/msg received
		go node.RunElectionTimer()
		return

	}
}

// StartElection is called when candidate is ready to start an election
func (node *RaftNode) StartElection() {

	var received_votes int32 = 1
	replica_id := int32(0)

	for _, client_obj := range node.peer_replica_clients {

		if replica_id == node.replica_id {
			replica_id++
			continue
		}

		go func(node *RaftNode, client_obj protos.ConsensusServiceClient, replica_id int32) {

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
				CandidateId:  node.replica_id,
				LastLogIndex: latestLogIndex,
				LastLogTerm:  latestLogTerm,
			}

			node.raft_node_mutex.RUnlock()
			// log.Printf("\nRUnLock in StartElection\n")

			//request vote and get reply
			response, err := client_obj.RequestVote(context.Background(), &args)

			node.raft_node_mutex.Lock()
			// log.Printf("\nLock in StartElection after response\n")
			if err == nil {

				// by the time the RPC call returns an answer, this replica might have already transitioned to another state.

				if node.state != Candidate {
					// log.Printf("\nUnlock in StartElection after response\n")
					node.raft_node_mutex.Unlock()
					return
				}

				if response.Term > node.currentTerm { // the response node has higher term than current one

					node.ToFollower(response.Term)
					// log.Printf("\nUnlock in StartElection after response\n")
					node.raft_node_mutex.Unlock()
					return

				} else if response.Term == node.currentTerm {

					if response.VoteGranted {

						votes := int(atomic.AddInt32(&received_votes, 1))

						if votes*2 > int(node.n_replicas) { // won the Election
							node.ToLeader()

							return
						}

					}

				}

			} else {

				log.Printf("\nError in requesting vote from replica %v: %v", replica_id, err.Error())
				// log.Printf("\nUnlock in StartElection after response\n")
				node.raft_node_mutex.Unlock()

			}

		}(node, client_obj, replica_id)

		replica_id++

	}

	node.electionTimerRunning = false
	go node.RunElectionTimer() // begin the timer during which this candidate waits for votes
}
