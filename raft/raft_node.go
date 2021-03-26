package raft

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"

	"github.com/krithikvaidya/distributed-dns/raft/protos"
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
	peer_replica_clients  []protos.ConsensusServiceClient // Client objects to send messages to other peers
	grpc_server           *grpc.Server                    // The gRPC server object
	raft_server           *http.Server                    // The HTTP server object for the Raft server
	kv_store_server       *http.Server                    // The HTTP server object for the KV store server[TODO]
	kvstore_addr          string                          // Stores the address of the local key value store
	raft_persistence_file string                          // File where the log, currentTerm, votedFor, commitIndex and lastApplied are persisted
	leaderAddress         string                          // Address of the last known leader
	nodeAddress           string                          // Address of our node
	latestClient          string                          // Address of client that made latest write request
	shutdown_chan         chan string                     // Channel indicating termination of given module.
	Master_ctx            context.Context                 // A context derived from the master context for graceful shutdown
	Master_cancel         context.CancelFunc              // The cancel function for the above master context
}

// Main struct storing different aspects of the replica and it's state
// Refer to figure 2 of the raft paper
type RaftNode struct {
	protos.UnimplementedConsensusServiceServer

	Meta *NodeMetadata

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

// Initialize the RaftNode (and NodeMetadata) objects. Also restores persisted raft state, if any.
func InitializeNode(n_replica int32, rid int, keyvalue_addr string) *RaftNode {

	// Initializes RaftNode and NodeMetadata

	raft_node := &RaftNode{

		trackMessage: make(map[string][]string),

		currentTerm: 0,
		votedFor:    -1,

		stopElectiontimer:  make(chan bool),
		electionResetEvent: make(chan bool),
		commitIndex:        -1,       // index of highest log entry known to be committed.
		lastApplied:        -1,       // index of highest log entry applied to state machine.
		state:              Follower, // all nodes are initialized as followers

		commits_ready: make(chan int32),
		storage:       NewStorage(),
	}

	meta := &NodeMetadata{

		n_replicas:           n_replica,
		replica_id:           int32(rid),
		peer_replica_clients: make([]protos.ConsensusServiceClient, n_replica),

		kvstore_addr:          keyvalue_addr,
		raft_persistence_file: keyvalue_addr[1:],

		shutdown_chan: make(chan string),
	}

	raft_node.Meta = meta

	if raft_node.storage.HasData(raft_node.Meta.raft_persistence_file) {

		raft_node.RestoreFromStorage(raft_node.storage)
		log.Printf("\nRestored Persisted Data:\n")
		log.Printf("\nRestored currentTerm: %v\nRestored votedFor: %v\nRestored log: %v\nRestored log length: %v\n", raft_node.currentTerm, raft_node.votedFor, raft_node.log, len(raft_node.log))

	} else {

		log.Printf("\nNo persisted data found.\n")

	}

	return raft_node

}

// Attempt to connect to the gRPC servers of all other replicas, and obtain the client stubs
func (node *RaftNode) ConnectToPeerReplicas(ctx context.Context, rep_addrs []string) {

	// Attempt to connect to the gRPC servers of all other replicas, and obtain the client stubs.
	// The clients for each corresponding server is stored in client_objs.
	client_objs := make([]protos.ConsensusServiceClient, node.Meta.n_replicas)

	// NOTE: even if the grpc Dial to a given server fails the first time, the client stub can still be obtained.
	// RPC requests using such client stubs will succeed when the connection can be established to
	// the gRPC server.

	for i := int32(0); i < node.Meta.n_replicas; i++ {

		if i == node.Meta.replica_id {
			continue
		}

		connxn, err := grpc.Dial(rep_addrs[i], grpc.WithInsecure())
		CheckErrorFatal(err) // there will NOT be an error if the gRPC server is down.

		// Obtain client stub
		cli := protos.NewConsensusServiceClient(connxn)

		client_objs[i] = cli
	}

	node.Meta.peer_replica_clients = client_objs

	// Check what the persisted state was (if any), and accordingly proceed
	node.GetLock("ConnectToPeerReplicas")
	defer node.ReleaseLock("ConnectToPeerReplicas")

	// suppose node dies before some commits have been applied to the state machine, then
	// we want to finish applying them.
	if node.commitIndex > node.lastApplied {
		node.commits_ready <- (node.commitIndex - node.lastApplied)
	}

	if node.state == Follower {

		go node.RunElectionTimer(ctx) // RunElectionTimer defined in election.go

	} else {

		// If node was a leader or candidate earlier, it will give up leadership/candidacy.
		// the new leader will soon send an heartbeat/appendentries, which will
		// automatically convert it back to follower and call the election timer.

	}

}

// Apply committed entries to our key-value store.
func (node *RaftNode) ApplyToStateMachine(ctx context.Context, testing bool) {

	for {

		select {

		case <-ctx.Done():
			if !testing {
				node.Meta.shutdown_chan <- "ApplyToStateMachine shutdown successful."
			}
			return

		case to_commit := <-node.commits_ready:

			log.Printf("\nApplyToStateMachine received commit(s)\n")

			node.GetLock("ApplyToStateMachine")

			var entries []protos.LogEntry

			// Get the entries that are uncommited and need to be applied.
			entries = node.log[node.lastApplied+1 : node.lastApplied+to_commit+1]
			applied := int32(0)
			halt_applying := false

			for _, entry := range entries {

				client := http.Client{}

				switch entry.Operation[0] {

				case "POST":

					formData := url.Values{
						"value": {entry.Operation[2]},
					}

					url := fmt.Sprintf("http://localhost%s/%s", node.Meta.kvstore_addr, entry.Operation[1])
					resp, err := http.PostForm(url, formData)

					if err != nil {

						log.Printf("\nError in http.PostForm in POST ApplyToStateMachine: %v\n", err)

						halt_applying = true
						break
					}

					resp.Body.Close()

				case "PUT":

					formData := url.Values{
						"value": {entry.Operation[2]},
					}

					req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("http://localhost%s/%s", node.Meta.kvstore_addr, entry.Operation[1]), bytes.NewBufferString(formData.Encode()))
					if err != nil {
						log.Printf("\nError in http.NewRequest in PUT ApplyToStateMachine: %v\n", err)

						halt_applying = true
						break

					}
					req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")

					resp, err := client.Do(req)
					if err != nil {
						log.Printf("\nError in client.Do in PUT ApplyToStateMachine: %v\n", err)

						halt_applying = true
						break

					}

					resp.Body.Close()

				case "DELETE":

					req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("http://localhost%s/%s", node.Meta.kvstore_addr, entry.Operation[1]), nil)
					if err != nil {
						log.Printf("\nError in http.NewRequest in DELETE ApplyToStateMachine: %v\n", err)

						halt_applying = true
						break

					}
					req.Header.Set("Content-Type", "application/x-www-form-urlencoded;")

					resp, err := client.Do(req)
					if err != nil {

						log.Printf("\nError in client.Do in DELETE ApplyToStateMachine: %v\n", err)
						halt_applying = true
						break

					}

					resp.Body.Close()

				case "NO-OP":
					log.Printf("\nNO-OP encountered, continuing...\n")

				default:
					log.Printf("\nFatal: Invalid operation: %v\n", entry.Operation[0])

				}

				if halt_applying {
					break
				}

				applied += 1
			}

			node.lastApplied = node.lastApplied + applied
			node.PersistToStorage()

			node.ReleaseLock("ApplyToStateMachine")

		}

	}
}
