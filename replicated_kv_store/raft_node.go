package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"

	"github.com/krithikvaidya/distributed-dns/replicated_kv_store/protos"
	"google.golang.org/grpc"
)

type RaftNodeState int32

const (
	Follower RaftNodeState = iota
	Candidate
	Leader
	Down
)

// Main struct storing different aspects of the replica and it's state
// Refer to figure 2 in the paper
type RaftNode struct {
	protos.UnimplementedConsensusServiceServer
	ready_chan           chan bool                       // Channel to signal whether the node is ready for operation
	n_replicas           int32                           // The number of replicas in the current replicated system
	replicas_ready       int32                           // number of replicas that have connected to this replica's gRPC server.
	replica_id           int32                           // The unique ID for the current replica
	peer_replica_clients []protos.ConsensusServiceClient // client objects to send messages to other peers
	raft_node_mutex      sync.RWMutex                    // The mutex for working with the RaftNode struct
	electionTimerRunning bool                            // will be true if the node is a follower and the election timer is running
	kvstore_addr         string                          // stores respective port on which local key value store is running
	commits_ready        chan int32                      // Channel to signal the number of items commited once commit has been made to the log.

	ready_chan           chan bool                       // Channel to signal whether the node is ready for operation
	n_replicas           int32                           // The number of replicas in the current replicated system
	replicas_ready       int32                           // number of replicas that have connected to this replica's gRPC server.
	replica_id           int32                           // The unique ID for the current replica
	peer_replica_clients []protos.ConsensusServiceClient // client objects to send messages to other peers
	raft_node_mutex      sync.RWMutex                    // The mutex for working with the RaftNode struct
	kvstore_addr         string                          // stores respective port on which local key value store is running
	commits_ready        chan int32                      // Channel to signal the number of items commited once commit has been made to the log.

	trackMessage map[string][]string // tracks messages sent by clients

	// States mentioned in figure 2 of the paper:

	// State to be maintained on all replicas (TODO: persist)
	currentTerm   int32             // Latest term server has seen
	votedFor      int32             // Candidate ID of the node that received vote from current node in the latest term
	log           []protos.LogEntry // The array of the log entry structs
	leaderAddress string
	nodeAddress   string
	// State to be maintained on all replicas (unpersisted)
	stopElectiontimer  chan bool     // Channel to signal for stopping the election timer for the node
	electionResetEvent chan bool     // Channel to signal for resetting the election timer for the node
	commitIndex        int32         // Index of the highest long entry known to be committed
	lastApplied        int32         // Index of the highest log entry applied to the state machine
	state              RaftNodeState // The current state of the node(eg. Candidate, Leader, etc)

	// State to be maintained on the leader (unpersisted)
	nextIndex  []int32 // Indices of the next log entry to send to each server
	matchIndex []int32 // Indices of highest log entry known to be replicated on each server

	storage    *Storage // Used for Persistence
	fileStored string   //Name of file where things are stored
}

func InitializeNode(n_replica int32, rid int, keyvalue_port string) *RaftNode {

	rn := &RaftNode{

		n_replicas:           n_replica,
		ready_chan:           make(chan bool),
		replicas_ready:       0,
		replica_id:           int32(rid),
		peer_replica_clients: make([]protos.ConsensusServiceClient, n_replica),
		state:                Follower, // all nodes are initialized as followers
		kvstore_addr:         keyvalue_port,
		commits_ready:        make(chan int32),

		trackMessage: make(map[string][]string),

		currentTerm: 0, // unpersisted
		votedFor:    -1,
		// log:         make([]LogEntry, 0), // initialized with fixed capacity of 10000, change later.

		stopElectiontimer:  make(chan bool),
		electionResetEvent: make(chan bool),
		commitIndex:        -1, // index of highest log entry known to be committed.
		lastApplied:        -1, // index of highest log entry applied to state machine.
		storage:            NewStorage(),
		fileStored:         keyvalue_port[1:],
	}

	if rn.storage.HasData(rn.fileStored) {

		rn.restoreFromStorage(rn.storage)
		log.Printf("\nRestored Persisted Data:\n")
		log.Printf("\nCurrent currentTerm: %v\nCurrent votedFor: %v\nCurrent log: %v\nCurrent log length: %v\n", rn.currentTerm, rn.votedFor, rn.log, len(rn.log))

	} else {

		log.Printf("\nNo persisted data found.\n")

	}

	return rn

}

func (node *RaftNode) ConnectToPeerReplicas(rep_addrs []string) {

	// Attempt to connect to the gRPC servers of all other replicas, and obtain the client stubs.
	// The clients for each corresponding server is stored in client_objs.
	client_objs := make([]protos.ConsensusServiceClient, node.n_replicas)

	// NOTE: even if the grpc Dial to a given server fails the first time, the client stub can still be obtained.
	// RPC requests using such client stubs will succeed when the connection can be established to
	// the gRPC server.

	for i := int32(0); i < node.n_replicas; i++ {

		if i == node.replica_id {
			continue
		}

		connxn, err := grpc.Dial(rep_addrs[i], grpc.WithInsecure())
		CheckErrorFatal(err) // there will NOT be an error if the gRPC server is down.

		// Obtain client stub
		cli := protos.NewConsensusServiceClient(connxn)

		client_objs[i] = cli
	}

	node.peer_replica_clients = client_objs

	// Check what the persisted state was (if any), and accordingly proceed
	node.raft_node_mutex.Lock()
	defer node.raft_node_mutex.Unlock()

	if node.state == Follower {

		go node.RunElectionTimer() // RunElectionTimer defined in election.go

	} else if node.state == Candidate {

		// If candidate, let it restart election. The timer for waiting
		// for votes from other replicas will be called in StartElection
		node.StartElection()

		// CHECK:
		// We don't call ToCandidate immediately here because we don't want to increment
		// currentTerm and node.state and node.votedFor are already set correctly, so
		// nothing to persist.

	} else if node.state == Leader {

		// if node was a leader before we want to give up leadership.
		// the new leader will soon send an heartbeat/appendentries, which will
		// automatically convert it back to follower and call the election timer.

	}

}

func (node *RaftNode) restoreFromStorage(storage *Storage) {
	if termvalue, check := node.storage.Get("currentTerm", node.fileStored); check {
		temp := gob.NewDecoder(bytes.NewBuffer(termvalue))
		temp.Decode(&node.currentTerm)
	} else {
		log.Fatalf("\nFatal: persisted data found, but currentTerm not found in storage\n")
	}
	if votedcheck, check := node.storage.Get("votedFor", node.fileStored); check {
		temp := gob.NewDecoder(bytes.NewBuffer(votedcheck))
		temp.Decode(&node.votedFor)
	} else {
		log.Fatalf("\nFatal: persisted data found, but currentTerm not found in storage\n")
	}
	if logentries, check := node.storage.Get("log", node.fileStored); check {
		temp := gob.NewDecoder(bytes.NewBuffer(logentries))
		temp.Decode(&node.log)
	} else {
		log.Fatalf("\nFatal: persisted data found, but currentTerm not found in storage\n")
	}
}

func (node *RaftNode) persistToStorage() {

	var termvalue bytes.Buffer
	gob.NewEncoder(&termvalue).Encode(node.currentTerm)
	node.storage.Set("currentTerm", termvalue.Bytes(), node.fileStored)

	var votedcheck bytes.Buffer
	gob.NewEncoder(&votedcheck).Encode(node.votedFor)
	node.storage.Set("votedFor", votedcheck.Bytes(), node.fileStored)

	var logentries bytes.Buffer
	gob.NewEncoder(&logentries).Encode(node.log)
	node.storage.Set("log", logentries.Bytes(), node.fileStored)

}

// Apply committed entries to our key-value store.
func (node *RaftNode) ApplyToStateMachine() {

	for {

		// log.Printf("\nIn ApplyToStateMachine\n")
		to_commit := <-node.commits_ready
		log.Printf("\nApplyToStateMachine received commit(s)\n")

		node.raft_node_mutex.Lock()

		var entries []protos.LogEntry

		// Get the entries that are uncommited and need to be applied.
		entries = node.log[node.lastApplied+1 : node.lastApplied+to_commit+1]

		for _, entry := range entries {

			client := http.Client{}

			switch entry.Operation[0] {

			case "POST":

				formData := url.Values{
					"value": {entry.Operation[2]},
				}

				url := fmt.Sprintf("http://localhost%s/%s", node.kvstore_addr, entry.Operation[1])
				resp, err := http.PostForm(url, formData)

				if err != nil {
					log.Printf("\nError in client.Do(req): %v\n", err)
					node.raft_node_mutex.Unlock()
					break
				}

				resp.Body.Close()

			case "PUT":

				formData := url.Values{
					"value": {entry.Operation[2]},
				}

				req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("http://localhost%s/%s", node.kvstore_addr, entry.Operation[1]), bytes.NewBufferString(formData.Encode()))
				if err != nil {
					log.Printf("\nError in http.NewRequest: %v\n", err)
					node.raft_node_mutex.Unlock()
					break
				}
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")

				resp, err := client.Do(req)
				if err != nil {
					log.Printf("\nError in client.Do: %v\n", err)
					node.raft_node_mutex.Unlock()
					break
				}

				resp.Body.Close()

			case "DELETE":

				req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("http://localhost%s/%s", node.kvstore_addr, entry.Operation[1]), nil)
				if err != nil {
					log.Printf("\nError in http.NewRequest: %v\n", err)
					node.raft_node_mutex.Unlock()
					break
				}
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded;")

				resp, err := client.Do(req)
				if err != nil {
					log.Printf("\nError in client.Do: %v\n", err)
					node.raft_node_mutex.Unlock()
					break
				}

				resp.Body.Close()

			case "NO-OP":
				log.Printf("\nNO-OP encountered, continuing...\n")

			default:
				log.Printf("\nFatal: Invalid operation: %v\n", entry.Operation[0])

			}

		}

		node.lastApplied = node.lastApplied + to_commit
		// log.Printf("Required Operations done to kv_store; Current lastApplied: %v", node.lastApplied)
		node.raft_node_mutex.Unlock()
	}
}
