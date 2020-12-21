package main

import (
	"context"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/krithikvaidya/distributed-dns/replicated_kv_store/protos"
)

// ToFollower is called when you get a term higher than your own
func (node *RaftNode) ToFollower(term int32) {
	node.state = Follower
	node.currentTerm = term
	node.votedFor = -1

	go node.RunElectionTimer()
	// Once converted to follower
	//we need to run the election timeout in the background
}

// ToCandidate is called when election timer runs out
// without heartbeat from leader
func (node *RaftNode) ToCandidate() {
	node.state = Candidate
	node.currentTerm++
	node.votedFor = node.replica_id

	node.StartElection()
	//we can start an election for the candidate to become the leader
}

// ToLeader is called when the candidate gets majority votes in election
func (node *RaftNode) ToLeader() {
	node.state = Leader

	go node.HeartBeats()
}

// ElectionStopper writes to the channel when election exit conditions are met
func (node *RaftNode) ElectionStopper(start int32) {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		<-ticker.C
		if node.state != Candidate && node.state != Follower {
			node.stopElectiontimer <- true
			return
		}

		if start != node.currentTerm {
			node.stopElectiontimer <- true
			return
		}
	}
}

// RunElectionTimer runs an election if no heartbeat is received
func (node *RaftNode) RunElectionTimer() {
	duration := time.Duration(150+rand.Intn(150)) * time.Millisecond
	//150 - 300 ms random time was mentioned in the paper

	start := node.currentTerm

	go node.ElectionStopper(start)

	select {
	case <-time.After(duration): //for timeout to call election
		node.ToCandidate()
		return
	case <-node.stopElectiontimer: //to stop timer
		return
	case <-node.electionResetEvent: //to reset timer when heartbeat/msg received
		time.Sleep(1 * time.Second)
	}
}

//HeartBeats is a goroutine that periodically makes leader
//send heartbeats as long as it is the leader
func (node *RaftNode) HeartBeats() {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		//{.....} call function to send hearbeats to all other nodes

		<-ticker.C
		if node.state != Leader {
			return
		}
	}
}

// StartElection is called when candidate is ready to start an election
func (node *RaftNode) StartElection() {
	//requestvote RPC
	//if election won call ToLeader
	//else call ToFollower
	saved_term := node.currentTerm
	var received_votes int32 = 1

	for _, client_obj := range node.peer_replica_clients {
		go func(client_obj protos.ConsensusServiceClient) {
			args := protos.RequestVoteMessage{
				Term:        saved_term,
				CandidateId: node.replica_id,
			}
			//request vote and get reply
			response, err := node.RequestVote(context.Background(), &args)
			if err != nil {
				if node.state != Candidate {
					return
				}

				if response.Term > saved_term { //The response node has higher term than current one
					node.ToFollower(response.Term)
					return
				} else if response.Term == saved_term {
					if response.VoteGranted {
						votes := int(atomic.AddInt32(&received_votes, 1))
						if votes*2 > n_replica { //Won the Election
							node.ToLeader()
							return
						}
					}
				}
			}
		}(client_obj)
	}
	go node.RunElectionTimer()
}
