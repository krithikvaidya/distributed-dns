package main

import (
<<<<<<< HEAD
	"log"
=======
	"bytes"
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"
>>>>>>> ff96df93e362d5940cc07b45ff9d908a46bec8b1

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

	log.Printf("\nUnlock in ToLeader()\n")
	node.raft_node_mutex.Unlock()

	success := make(chan bool)
	node.LeaderSendAEs("NO-OP", msg, int32(len(node.log)-1), success)
	<-success

	go node.HeartBeats()
}

func (node *RaftNode) Writeto() {
	for range node.newCommitReadyChan {
		// Find which entries we have to apply.
		node.raft_node_mutex.Lock()
		//savedTerm := node.currentTerm
		//savedLastApplied := node.lastApplied
		var entries []protos.LogEntry
		if node.commitIndex > node.lastApplied {
			entries = node.log[node.lastApplied+1 : node.commitIndex+1]
			node.lastApplied = node.commitIndex
			log.Printf("Operations to be applied to kv_store\n")
		} else {
			return
		}
		node.raft_node_mutex.Unlock()

		for _, entry := range entries {
			formData := url.Values{
				"value": {entry.Operation[2]},
			}

			timeout := time.Duration(100 * time.Microsecond)
			client := http.Client{
				Timeout: timeout,
			}
			Oper := strings.ToLower(entry.Operation[0])
			switch Oper {
			case "push":
				formData := url.Values{
					"value": {entry.Operation[2]},
				}

				req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:%d/%s", node.storePort, entry.Operation[1]), bytes.NewBufferString(formData.Encode()))
				CheckError(err)
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")

				resp, err := client.Do(req)
				CheckError(err)
				defer resp.Body.Close()
				break

			case "put":

				req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("http://localhost:%d/%s", node.storePort, entry.Operation[1]), bytes.NewBufferString(formData.Encode()))
				CheckError(err)
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")

				resp, err := client.Do(req)
				CheckError(err)
				defer resp.Body.Close()
				break

			case "delete":
				req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("http://localhost:%d/%s", node.storePort, entry.Operation[1]), bytes.NewBufferString(""))
				CheckError(err)

				resp, err := client.Do(req)
				CheckError(err)
				defer resp.Body.Close()
				break

			case "no-op":
				break
			}

		}
		log.Println("Required Operations done to kv_store; In-sync")
	}
}
