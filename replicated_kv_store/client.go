package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/krithikvaidya/distributed-dns/replicated_kv_store/protos"
)

//WriteCommand is called when the client sends the replica a write request.
func (node *RaftNode) WriteCommand(operation []string) bool {
	for {
		if node.commitIndex == node.lastApplied {
			// Perform operation only if leader
			if node.state == Leader {

				//append to local log
				node.log = append(node.log, protos.LogEntry{Term: node.currentTerm, Operation: operation})

				// log.Printf("\nnode.log.operation: %v\n", node.log[len(node.log)-1].Operation)

				var entries []*protos.LogEntry
				entries = append(entries, &node.log[len(node.log)-1])

				msg := &protos.AppendEntriesMessage{

					Term:         node.currentTerm,
					LeaderId:     node.replica_id,
					PrevLogIndex: int32(len(node.log) - 1),
					PrevLogTerm:  node.log[len(node.log)-1].Term,
					LeaderCommit: node.commitIndex,
					Entries:      entries,
				}

				successful_write := make(chan bool)

				node.LeaderSendAEs(operation[0], msg, int32(len(node.log)-1), successful_write)

				node.raft_node_mutex.Unlock() // Lock was acquired in the respective calling Handler function in raft_server.go

				success := <-successful_write //Written to from AE when majority of nodes have replicated the write or failure occurs

				if success {
					node.raft_node_mutex.Lock()
					node.commitIndex++
					node.raft_node_mutex.Unlock()
					node.commits_ready <- 1
					log.Printf("\nWrite operation successfully completed and committed.\n")
				} else {
					log.Printf("\nWrite operation failed.\n")
				}

				return success
			}

		if success {
			node.raft_node_mutex.Lock()
			node.commitIndex++
			node.persistToStorage()
			node.raft_node_mutex.Unlock()
			return false
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// ReadCommand is different since read operations do not need to be added to log
func (node *RaftNode) ReadCommand(key string) (string, error) {
	for {
		if node.commitIndex != node.lastApplied {

			write_success := make(chan bool)
			node.StaleReadCheck(write_success)
			node.raft_node_mutex.RUnlock()
			status := <-write_success

			node.raft_node_mutex.RLock()
			defer node.raft_node_mutex.RUnlock()

			if (status == true) && (node.state == Leader) {

				// assuming that if an operation on the state machine succeeds on one of the replicas,
				// it will succeed on all. and vice versa.
				url := fmt.Sprintf("http://localhost%s/%s", node.kvstore_addr, key)

				resp, err := http.Get(url)

				if err == nil {

					defer resp.Body.Close()
					contents, err2 := ioutil.ReadAll(resp.Body)

					if err2 != nil {
						log.Printf(Red + "[Error]" + Reset + ": " + err2.Error())
						return "unable to perform read", err2

					}

					log.Printf("\nREAD successful.\n")

					return string(contents), nil

				} else {

					log.Printf(Red + "[Error]" + Reset + ": " + err.Error())

				}
			}

			return "unable to perform read", errors.New("read_failed")
		}
	}
}

// StaleReadCheck sends dummy heartbeats to make sure that a new leader has not come
func (node *RaftNode) StaleReadCheck(write_success chan bool) {
	replica_id := 0

	var entries []*protos.LogEntry

	prevLogIndex := node.nextIndex[replica_id] - 1
	prevLogTerm := int32(-1)

	if prevLogIndex >= 0 {
		prevLogTerm = node.log[prevLogIndex].Term
	}

	hbeat_msg := &protos.AppendEntriesMessage{

		Term:         node.currentTerm,
		LeaderId:     node.replica_id,
		PrevLogIndex: prevLogIndex,
		PrevLogTerm:  prevLogTerm,
		LeaderCommit: node.commitIndex,
		Entries:      entries,
	}

	node.LeaderSendAEs("HBEAT", hbeat_msg, int32(len(node.log)-1), write_success)
}
