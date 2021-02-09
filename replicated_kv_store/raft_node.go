package main

import (
	"bytes"
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

// Store metadata related to the key value store and the raft node.
type NodeMetadata struct {
	n_replicas            int32                           // The number of replicas in the current replicated system
	replica_id            int32                           // The unique ID for the current replica
	peer_replica_clients  []protos.ConsensusServiceClient // client objects to send messages to other peers
	grpc_server           *grpc.Server                    // The gRPC server object
	raft_server           *http.Server                    // The HTTP server object for the Raft server
	kv_store_server       *http.Server                    // The HTTP server object for the KV store server[TODO]
	kvstore_addr          string                          // Stores the address of the local key value store
	raft_persistence_file string                          // File where the log, currentTerm, votedFor, commitIndex and lastApplied are persisted
	leaderAddress         string                          // Address of the last known leader
	nodeAddress           string                          // Address of our node
	latestClient          string                          // Address of client that made latest write request

}

// Main struct storing different aspects of the replica and it's state
// Refer to figure 2 in the paper
type RaftNode struct {
	protos.UnimplementedConsensusServiceServer

	node_meta *NodeMetadata

	raft_node_mutex sync.RWMutex // The mutex for working with the RaftNode struct

	trackMessage map[string][]string // tracks messages sent by clients

	// States mentioned in figure 2 of the paper:

	// State to be maintained on all replicas
	currentTerm int32             // Latest term server has seen
	votedFor    int32             // Candidate ID of the node that received vote from current node in the latest term
	log         []protos.LogEntry // The array of the log entry structs

	// State to be maintained on all replicas
	stopElectiontimer  chan bool     // Channel to signal for stopping the election timer for the node
	electionResetEvent chan bool     // Channel to signal for resetting the election timer for the node
	commitIndex        int32         // Index of the highest long entry known to be committed. Persisted.
	lastApplied        int32         // Index of the highest log entry applied to the state machine. Persisted.
	state              RaftNodeState // The current state of the node(eg. Candidate, Leader, etc)

	// State to be maintained on the leader (unpersisted)
	nextIndex  []int32 // Indices of the next log entry to send to each server
	matchIndex []int32 // Indices of highest log entry known to be replicated on each server

	commits_ready chan int32 // Channel to signal the number of items commited once commit has been made to the log.
	storage       *Storage   // Used for Persistence
}

func InitializeNode(n_replica int32, rid int, keyvalue_addr string) *RaftNode {

	// Initializes RaftNode and NodeMetadata

	raft_node := &RaftNode{

		trackMessage: make(map[string][]string),

		currentTerm: 0, // unpersisted
		votedFor:    -1,
		// log:         make([]LogEntry, 0), // initialized with fixed capacity of 10000, change later.

		stopElectiontimer:  make(chan bool),
		electionResetEvent: make(chan bool),
		commitIndex:        -1,       // index of highest log entry known to be committed.
		lastApplied:        -1,       // index of highest log entry applied to state machine.
		state:              Follower, // all nodes are initialized as followers

		commits_ready: make(chan int32),
		storage:       NewStorage(),
	}

	node_meta := &NodeMetadata{

		n_replicas:           n_replica,
		replica_id:           int32(rid),
		peer_replica_clients: make([]protos.ConsensusServiceClient, n_replica),

		kvstore_addr:          keyvalue_addr,
		raft_persistence_file: keyvalue_addr[1:],
	}

	raft_node.node_meta = node_meta

	if raft_node.storage.HasData(raft_node.node_meta.raft_persistence_file) {

		raft_node.restoreFromStorage(raft_node.storage)
		log.Printf("\nRestored Persisted Data:\n")
		log.Printf("\nCurrent currentTerm: %v\nCurrent votedFor: %v\nCurrent log: %v\nCurrent log length: %v\n", raft_node.currentTerm, raft_node.votedFor, raft_node.log, len(raft_node.log))

	} else {

		log.Printf("\nNo persisted data found.\n")

	}

	return raft_node

}

func (node *RaftNode) ConnectToPeerReplicas(rep_addrs []string) {

	// Attempt to connect to the gRPC servers of all other replicas, and obtain the client stubs.
	// The clients for each corresponding server is stored in client_objs.
	client_objs := make([]protos.ConsensusServiceClient, node.node_meta.n_replicas)

	// NOTE: even if the grpc Dial to a given server fails the first time, the client stub can still be obtained.
	// RPC requests using such client stubs will succeed when the connection can be established to
	// the gRPC server.

	for i := int32(0); i < node.node_meta.n_replicas; i++ {

		if i == node.node_meta.replica_id {
			continue
		}

		connxn, err := grpc.Dial(rep_addrs[i], grpc.WithInsecure())
		CheckErrorFatal(err) // there will NOT be an error if the gRPC server is down.

		// Obtain client stub
		cli := protos.NewConsensusServiceClient(connxn)

		client_objs[i] = cli
	}

	node.node_meta.peer_replica_clients = client_objs

	// Check what the persisted state was (if any), and accordingly proceed
	node.raft_node_mutex.Lock()
	defer node.raft_node_mutex.Unlock()

	// suppose node dies before that series of commits have been applied then we want to finish it
	if node.commitIndex > node.lastApplied {
		node.commits_ready <- (node.commitIndex - node.lastApplied)
	}

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

	var check bool

	var t1, t2, t3, t4, t5 interface{}

	if t1, check = node.storage.Get("currentTerm", node.node_meta.raft_persistence_file); !check {
		log.Fatalf("\nFatal: persisted data found, but currentTerm not found in storage\n")
	}

	node.currentTerm = t1.(int32)

	if t2, check = node.storage.Get("votedFor", node.node_meta.raft_persistence_file); !check {
		log.Fatalf("\nFatal: persisted data found, but votedFor not found in storage\n")
	}

	node.votedFor = t2.(int32)

	if t3, check = node.storage.Get("log", node.node_meta.raft_persistence_file); !check {
		log.Fatalf("\nFatal: persisted data found, but log not found in storage\n")
	}

	node.log = t3.([]protos.LogEntry)

	if t4, check = node.storage.Get("commitIndex", node.node_meta.raft_persistence_file); !check {
		log.Fatalf("\nFatal: persisted data found, but commitIndex not found in storage\n")
	}

	node.commitIndex = t4.(int32)

	if t5, check = node.storage.Get("lastApplied", node.node_meta.raft_persistence_file); !check {
		log.Fatalf("\nFatal: persisted data found, but lastApplied not found in storage\n")
	}

	node.lastApplied = t5.(int32)
}

func (node *RaftNode) persistToStorage() {

	node.storage.Set("currentTerm", node.currentTerm)
	node.storage.Set("votedFor", node.votedFor)
	node.storage.Set("log", node.log)
	node.storage.Set("commitIndex", node.commitIndex)
	node.storage.Set("lastApplied", node.lastApplied)

	node.storage.WriteFile(node.node_meta.raft_persistence_file)

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

				url := fmt.Sprintf("http://localhost%s/%s", node.node_meta.kvstore_addr, entry.Operation[1])
				resp, err := http.PostForm(url, formData)

				if err != nil {
					log.Printf("\nError in client.Do(req): %v\n", err)
					continue
				}

				resp.Body.Close()

			case "PUT":

				formData := url.Values{
					"value": {entry.Operation[2]},
				}

				req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("http://localhost%s/%s", node.node_meta.kvstore_addr, entry.Operation[1]), bytes.NewBufferString(formData.Encode()))
				if err != nil {
					log.Printf("\nError in http.NewRequest: %v\n", err)
					continue
				}
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")

				resp, err := client.Do(req)
				if err != nil {
					log.Printf("\nError in client.Do: %v\n", err)
					continue
				}

				resp.Body.Close()

			case "DELETE":

				req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("http://localhost%s/%s", node.node_meta.kvstore_addr, entry.Operation[1]), nil)
				if err != nil {
					log.Printf("\nError in http.NewRequest: %v\n", err)
					continue
				}
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded;")

				resp, err := client.Do(req)
				if err != nil {
					log.Printf("\nError in client.Do: %v\n", err)
					continue
				}

				resp.Body.Close()

			case "NO-OP":
				log.Printf("\nNO-OP encountered, continuing...\n")

			default:
				log.Printf("\nFatal: Invalid operation: %v\n", entry.Operation[0])

			}

		}

		node.lastApplied = node.lastApplied + to_commit
		node.persistToStorage()
		// log.Printf("Required Operations done to kv_store; Current lastApplied: %v", node.lastApplied)
		node.raft_node_mutex.Unlock()
	}
}
