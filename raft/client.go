package raft

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"reflect"
	"time"

	"github.com/krithikvaidya/distributed-dns/raft/protos"
)

/*
WriteCommand is called by the HTTP handler functions when the client sends the replica
a write (POST/PUT/DELETE) request. It parses the command and calls LeaderSendAEs for
performing the write. If the write request is successfully persisted across a majority
of replicas, it informs ApplyToStateMachine to perform the write operation on the key-value store.
*/
func (node *RaftNode) WriteCommand(operation []string, client string) (bool, error) {

	for node.commitIndex != node.lastApplied {

		node.raft_node_mutex.RUnlock() // Lock was acquired in the respective calling Handler function in raft_server.go
		time.Sleep(20 * time.Millisecond)
		node.raft_node_mutex.RLock()

	}

	// TODO: make the below upgrade of the mutex lock atomic.
	node.raft_node_mutex.RUnlock()
	node.raft_node_mutex.Lock()

	if node.state != Leader {
		defer node.raft_node_mutex.Unlock()
		return false, errors.New("\nNot a leader.\n")
	}

	var equal bool
	var Err error

	lastClientOper, val := node.trackMessage[client] //lastClientOper is the operation done by the given client previously

	/**
	* We do not want to accept duplicate PUT/POST/DELETE requests from clients.
	* In case the latest request that the raft system has received is the same
	* as the most latest received request, and the client for that request is the
	* same, then we reject it.
	* TODO: implement the functionality for DELETE. DELETE requests are not allowed to
	* have a request body.
	 */
	if val && operation[0] != "DELETE" {
		equal = reflect.DeepEqual(lastClientOper, operation)
		equal = equal && client == node.Meta.latestClient
	}

	if equal {
		Err = errors.New("Write operation failed. Already received identical write request from identical clientid.")
		defer node.raft_node_mutex.Unlock()
		return false, Err
	}

	// If it's a PUT or DELETE request, ensure that the resource exists.
	if operation[0] == "PUT" || operation[0] == "DELETE" {
		
		node.raft_node_mutex.Unlock()
		node.raft_node_mutex.RLock()
		response, err := node.ReadCommand(operation[1])

		if err == nil && response == "Invalid key value pair\n" {
			prnt_str := fmt.Sprintf("\nUnable to perform %v request, no value exists for given key in the store.\n", operation[0])
			return false, errors.New(prnt_str)
		}

		node.raft_node_mutex.RUnlock()
		node.raft_node_mutex.Lock()
	}

	node.Meta.latestClient = client

	var entries []*protos.LogEntry
	entries = append(entries, &node.log[len(node.log)-1])

	msg := &protos.AppendEntriesMessage{

		Term:         node.currentTerm,
		LeaderId:     node.Meta.replica_id,
		LeaderCommit: node.commitIndex,
		LatestClient: node.Meta.latestClient,
		LeaderAddr:   node.Meta.nodeAddress,
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
		node.trackMessage[client] = operation
		node.persistToStorage()
		node.raft_node_mutex.Unlock()
		node.commits_ready <- 1

	} else {
		Err = errors.New("Write operation failed. Write could not be replicated on majority of nodes.")
	}

	return success, Err

}

/*
ReadCommand is called when the client sends the replica a read request.
Read operations do not need to be added to the log.
*/
func (node *RaftNode) ReadCommand(key string) (string, error) {
	for node.commitIndex != node.lastApplied {
		node.raft_node_mutex.RUnlock()
		time.Sleep(20 * time.Millisecond)
		node.raft_node_mutex.RLock()
	}

	heartbeat_success := make(chan bool)
	node.StaleReadCheck(heartbeat_success)
	node.raft_node_mutex.RUnlock()
	status := <-heartbeat_success

	node.raft_node_mutex.RLock()

	if (status == true) && (node.state == Leader) {

		url := fmt.Sprintf("http://localhost%s/%s", node.Meta.kvstore_addr, key)

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

// StaleReadCheck sends dummy heartbeats to make sure that a new leader has not been elected.
func (node *RaftNode) StaleReadCheck(heartbeat_success chan bool) {

	hbeat_msg := &protos.AppendEntriesMessage{

		Term:         node.currentTerm,
		LeaderId:     node.Meta.replica_id,
		LeaderCommit: node.commitIndex,
		LatestClient: node.Meta.latestClient,
		LeaderAddr:   node.Meta.nodeAddress,
	}

	node.LeaderSendAEs("HBEAT", hbeat_msg, int32(len(node.log)-1), heartbeat_success)
}
