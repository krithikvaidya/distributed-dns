package main

import (
	"context"
	"log"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/krithikvaidya/distributed-dns/replicated_kv_store/protos"
)

// ToFollower is called when you get a term higher than your own
func (node *RaftNode) ToFollower(term int32) {

	prevState := node.state
	node.state = Follower
	node.currentTerm = term
	node.votedFor = -1

	// If node was a leader, start election timer. Else if it was a candidate, reset the election timer.
	if prevState == Leader {
		node.electionTimerRunning = true
		go node.RunElectionTimer()
	} else {
		if node.electionTimerRunning {
			node.electionResetEvent <- true
		}
	}

}

// ToCandidate is called when election timer runs out
// without heartbeat from leader
func (node *RaftNode) ToCandidate() {

	node.state = Candidate
	node.currentTerm++
	node.votedFor = node.replica_id

	//we can start an election for the candidate to become the leader
	node.StartElection()
}

// ToLeader is called when the candidate gets majority votes in election
func (node *RaftNode) ToLeader() {

	log.Printf("\nTransitioned to leader\n")
	// stop election timer since leader doesn't need it
	node.stopElectiontimer <- true
	node.electionTimerRunning = false

	node.state = Leader

	node.nextIndex = make([]int32, node.n_replicas, node.n_replicas)
	node.matchIndex = make([]int32, node.n_replicas, node.n_replicas)

	// initialize nextIndex, matchIndex
	for replica_id := 0; replica_id < len(node.peer_replica_clients); replica_id++ {

		if int32(replica_id) == node.replica_id {
			continue
		}

		node.nextIndex[replica_id] = int32(len(node.log))
		node.matchIndex[replica_id] = int32(0)

	}

	// send no-op for synchronization
	// first obtain prevLogIndex and prevLogTerm

	prevLogIndex := int32(-1)
	prevLogTerm := int32(-1)

	if logLen := int32(len(node.log)); logLen > 0 {
		prevLogIndex = logLen - 1
		prevLogTerm = node.log[prevLogIndex].Term
	}

	var operation []string
	operation = append(operation, "NO-OP")

	node.log = append(node.log, protos.LogEntry{Term: node.currentTerm, Operation: operation})

	var entries []*protos.LogEntry
	entries = append(entries, &node.log[len(node.log)-1])

	msg := &protos.AppendEntriesMessage{

		Term:         node.currentTerm,
		LeaderId:     node.replica_id,
		PrevLogIndex: prevLogIndex,
		PrevLogTerm:  prevLogTerm,
		LeaderCommit: node.commitIndex,
		Entries:      entries,
	}

	node.LeaderSendAEs("NO-OP", msg, int32(len(node.log)-1))

	go node.HeartBeats()
}

// RunElectionTimer runs an election if no heartbeat is received
func (node *RaftNode) RunElectionTimer() {
	duration := time.Duration(150+rand.Intn(150)) * time.Millisecond
	//150 - 300 ms random time was mentioned in the paper

	// go node.ElectionStopper(start)

	select {

	case <-time.After(duration): //for timeout to call election

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
		return

	case <-node.electionResetEvent: //to reset timer when heartbeat/msg received
		go node.RunElectionTimer()
		return

	}
}

// To send AppendEntry to single replica, and retry if needed.
func (node *RaftNode) LeaderSendAE(replica_id int32, upper_index int32, client_obj protos.ConsensusServiceClient, msg *protos.AppendEntriesMessage) (status bool) {

	response, err := client_obj.AppendEntries(context.Background(), msg)

	if err != nil {
		// ...
	}

	if response.Success == false {

		if node.state != Leader {
			return
		}

		if response.Term > node.currentTerm {

			node.ToFollower(response.Term)
			return false
		}

		// will reach here if response.Term <= node.currentTerm and response.Success == false

		node.nextIndex[replica_id]--

		var entries []*protos.LogEntry

		for i := msg.PrevLogIndex; i <= upper_index; i++ {
			entries = append(entries, &node.log[i])
		}

		prevLogIndex := int32(msg.PrevLogIndex - 1)
		prevLogTerm := int32(-1)

		if prevLogIndex >= 0 {
			prevLogTerm = node.log[prevLogIndex].Term
		}

		new_msg := &protos.AppendEntriesMessage{

			Term:         node.currentTerm,
			LeaderId:     node.replica_id,
			PrevLogIndex: prevLogIndex,
			PrevLogTerm:  prevLogTerm,
			LeaderCommit: node.commitIndex,
			Entries:      entries,
		}

		return node.LeaderSendAE(replica_id, upper_index, client_obj, new_msg)

	} else {

		node.nextIndex[replica_id] = upper_index + 1
		node.matchIndex[replica_id] = upper_index

		return true

	}

}

// Leader sending AppendEntries to all other replicas.
func (node *RaftNode) LeaderSendAEs(msg_type string, msg *protos.AppendEntriesMessage, upper_index int32) {

	replica_id := int32(0)

	// node.raft_node_mutex.RLock() // for node.peer_replica_clients.
	// defer node.raft_node_mutex.RUnlock()
	// (The above are not required, since in the current implementation it doesnt change after initialization)

	successes := int32(1)

	for _, client_obj := range node.peer_replica_clients {

		if replica_id == node.replica_id {
			replica_id++
			continue
		}

		go func(node *RaftNode, client_obj protos.ConsensusServiceClient, replica_id int32, upper_index int32) {

			node.raft_node_mutex.Lock()
			// log.Printf("\nLocked in LeaderSendAEs spawned goroutine\n")
			// log.Printf("\nDATA: \n")
			// log.Printf("\nreplica_id: %v, upper_index: %v, msg: %v\n", replica_id, upper_index, msg)
			if node.LeaderSendAE(replica_id, upper_index, client_obj, msg) {
				tot_success := atomic.AddInt32(&successes, 1)
				if (msg_type != "HBEAT") && (tot_success > (node.n_replicas)/2) {
					log.Printf("node.commit_chan true")
					// node.commit_chan <- true
				}
			} else {
				log.Printf("node.commit_chan false")
				// node.commit_chan <- false
			}

			// log.Printf("\nUnLocked in LeaderSendAEs spawned goroutine\n")
			node.raft_node_mutex.Unlock()

		}(node, client_obj, replica_id, upper_index)

		replica_id++

	}

}

//HeartBeats is a goroutine that periodically makes leader
//send heartbeats as long as it is the leader
func (node *RaftNode) HeartBeats() {

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {

		<-ticker.C

		node.raft_node_mutex.RLock()
		// log.Printf("\nRLock in HeartBeats\n")

		if node.state != Leader {

			// log.Printf("\nRunLock in HeartBeats\n")
			node.raft_node_mutex.RUnlock()
			return
		}

		replica_id := 0

		prevLogIndex := node.nextIndex[replica_id] - 1
		prevLogTerm := int32(-1)

		if prevLogIndex >= 0 {
			prevLogTerm = node.log[prevLogIndex].Term
		}

		// send heartbeat
		var entries []*protos.LogEntry

		hbeat_msg := &protos.AppendEntriesMessage{

			Term:         node.currentTerm,
			LeaderId:     node.replica_id,
			PrevLogIndex: prevLogIndex,
			PrevLogTerm:  prevLogTerm,
			LeaderCommit: node.commitIndex,
			Entries:      entries,
		}

		node.LeaderSendAEs("HBEAT", hbeat_msg, int32(len(node.log)))
		// log.Printf("\nRunLock in HeartBeats\n")
		node.raft_node_mutex.RUnlock()

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
			log.Printf("\nRLock in StartElection\n")

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
			log.Printf("\nRUnLock in StartElection\n")

			//request vote and get reply
			response, err := client_obj.RequestVote(context.Background(), &args)

			node.raft_node_mutex.Lock()
			log.Printf("\nLock in StartElection after response\n")
			if err == nil {

				// by the time the RPC call returns an answer, this replica might have already transitioned to another state.

				if node.state != Candidate {
					log.Printf("\nUnlock in StartElection after response\n")
					node.raft_node_mutex.Unlock()
					return
				}

				if response.Term > node.currentTerm { // the response node has higher term than current one

					node.ToFollower(response.Term)
					log.Printf("\nUnlock in StartElection after response\n")
					node.raft_node_mutex.Unlock()
					return

				} else if response.Term == node.currentTerm {

					if response.VoteGranted {

						votes := int(atomic.AddInt32(&received_votes, 1))

						if votes*2 > int(node.n_replicas) { // won the Election
							node.ToLeader()
							log.Printf("\nUnlock in StartElection after response\n")
							node.raft_node_mutex.Unlock()
							return
						}

					}

				}

			} else {

				log.Printf("\nError in requesting vote from replica %v: %v", replica_id, err.Error())
				log.Printf("\nUnlock in StartElection after response\n")
				node.raft_node_mutex.Unlock()

			}

		}(node, client_obj, replica_id)

		replica_id++

	}

	go node.RunElectionTimer() // begin the timer during which this candidate waits for votes
	node.electionTimerRunning = false
}
