package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"reflect"
	"time"

	"github.com/krithikvaidya/distributed-dns/replicated_kv_store/protos"
)

//WriteCommand is called when the client sends the replica a write request.
func (node *RaftNode) WriteCommand(operation []string, client string) (bool, error) {

	for node.commitIndex != node.lastApplied {
		node.raft_node_mutex.RUnlock()
		time.Sleep(20 * time.Millisecond)
		node.raft_node_mutex.RLock()
	}
	node.raft_node_mutex.RUnlock() // Lock was acquired in the respective calling Handler function in raft_server.go
	node.raft_node_mutex.Lock()

	if node.state != Leader {
		node.raft_node_mutex.Unlock()
		return false, errors.New("\nNot a leader.\n")
	}

	var equal bool
	var Err error

	lastClientOper, val := node.trackMessage[client] //lastClientOper is the operation done by the given client previously

	//check if entry exists; if it does check if its the same as the one that was previously
	if val && operation[0] != "DELETE" {
		equal = reflect.DeepEqual(lastClientOper, operation)
		equal = equal && client == node.node_meta.latestClient
	}

	if equal {
		Err = errors.New("Write operation failed. Already received identical write request from identical clientid.")
		node.raft_node_mutex.Unlock()
		return false, Err
	}

	// If it's a PUT or DELETE request, ensure that the resource exists.
	if operation[0] == "PUT" || operation[0] == "DELETE" {

		node.raft_node_mutex.Unlock()
		node.raft_node_mutex.RLock()
		response, err := node.ReadCommand(operation[1])

		log.Printf("ok" + response + "ok")
		log.Printf("%v", (response == "Invalid key value pair\n"))

		if err == nil && response == "Invalid key value pair\n" {
			prnt_str := fmt.Sprintf("\nUnable to perform %v request, no value exists for given key in the store.\n", operation[0])
			return false, errors.New(prnt_str)
		}

		node.raft_node_mutex.Lock()
	}

	node.node_meta.latestClient = client

	var entries []*protos.LogEntry
	entries = append(entries, &node.log[len(node.log)-1])

	msg := &protos.AppendEntriesMessage{

		Term:         node.currentTerm,
		LeaderId:     node.node_meta.replica_id,
		LeaderCommit: node.commitIndex,
		LatestClient: node.node_meta.latestClient,
		LeaderAddr:   node.node_meta.nodeAddress,
	}

	//append to local log
	node.log = append(node.log, protos.LogEntry{Term: node.currentTerm, Operation: operation, Clientid: client})

	successful_write := make(chan bool)

	node.LeaderSendAEs(operation[0], msg, int32(len(node.log)-1), successful_write)

	node.raft_node_mutex.Unlock()

	success := <-successful_write //Written to from AE when majority of nodes have replicated the write or failure occurs

	if success {

		node.raft_node_mutex.Lock()
		node.commitIndex++
		node.persistToStorage()
		node.raft_node_mutex.Unlock()
		node.commits_ready <- 1

		node.trackMessage[client] = operation

	} else {
		Err = errors.New("Write operation failed. Write could not be replicated on majority of nodes.")
	}

	return success, Err

}

// ReadCommand is different since read operations do not need to be added to log
func (node *RaftNode) ReadCommand(key string) (string, error) {
	for node.commitIndex != node.lastApplied {
		node.raft_node_mutex.RUnlock()
		time.Sleep(20 * time.Millisecond)
		node.raft_node_mutex.RLock()
	}

	write_success := make(chan bool)
	node.StaleReadCheck(write_success)
	node.raft_node_mutex.RUnlock()
	status := <-write_success

	node.raft_node_mutex.RLock()
	defer node.raft_node_mutex.RUnlock()

	if (status == true) && (node.state == Leader) {

		// assuming that if an operation on the state machine succeeds on one of the replicas,
		// it will succeed on all. and vice versa.
		url := fmt.Sprintf("http://localhost%s/%s", node.node_meta.kvstore_addr, key)

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

// StaleReadCheck sends dummy heartbeats to make sure that a new leader has not come
func (node *RaftNode) StaleReadCheck(write_success chan bool) {

	hbeat_msg := &protos.AppendEntriesMessage{

		Term:         node.currentTerm,
		LeaderId:     node.node_meta.replica_id,
		LeaderCommit: node.commitIndex,
		LatestClient: node.node_meta.latestClient,
		LeaderAddr:   node.node_meta.nodeAddress,
	}

	node.LeaderSendAEs("HBEAT", hbeat_msg, int32(len(node.log)-1), write_success)
}
