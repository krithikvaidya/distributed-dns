package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
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
	electionTimerRunning bool
	kvstore_addr         string     //to store respective port on which replicated key value store is running
	commits_ready        chan int32 // Channel to signal once commit has been made

	// States mentioned in figure 2 of the paper:

	// State to be maintained on all replicas (TODO: persist)
	currentTerm int32             // Latest term server has seen
	votedFor    int32             // Candidate ID of the node that received vote from current node in the latest term
	log         []protos.LogEntry // The array of the log entry structs

	// State to be maintained on all replicas
	stopElectiontimer  chan bool     // Channel to signal for stopping the election timer for the node
	electionResetEvent chan bool     // Channel to signal for resetting the election timer for the node
	commitIndex        int32         // Index of the highest long entry known to be committed
	lastApplied        int32         // Index of the highest log entry applied to the state machine
	state              RaftNodeState // The current state of the node(eg. Candidate, Leader, etc)

	// State to be maintained on the leader
	nextIndex  []int32 // Indices of the next log entry to send to each server
	matchIndex []int32 // Indices of highest log entry known to be replicated on each server
}

func InitializeNode(n_replica int32, rid int32, keyvalue_port string) *RaftNode {

	rn := &RaftNode{

		n_replicas:           n_replica,
		ready_chan:           make(chan bool),
		replicas_ready:       0,
		replica_id:           rid,
		peer_replica_clients: make([]protos.ConsensusServiceClient, n_replica),
		state:                Follower, // all nodes are initialized as followers
		electionTimerRunning: false,
		kvstore_addr:         keyvalue_port,
		commits_ready:        make(chan int32),

		currentTerm: 0, // unpersisted
		votedFor:    -1,
		// log:         make([]LogEntry, 0), // initialized with fixed capacity of 10000, change later.

		stopElectiontimer:  make(chan bool),
		electionResetEvent: make(chan bool),
		commitIndex:        -1, // index of highest log entry known to be committed.
		lastApplied:        -1, // index of highest log entry applied to state machine.
	}

	return rn

}

func (node *RaftNode) ConnectToPeerReplicas(rep_addrs []string) {

	// Attempt to connect to the gRPC servers of all other replicas
	// The clients for each corresponding server is stored in client_objs
	client_objs := make([]protos.ConsensusServiceClient, node.n_replicas)

	for i := int32(0); i < node.n_replicas; i++ {

		if i == node.replica_id {
			continue
		}

		connxn, err := grpc.Dial(rep_addrs[i], grpc.WithInsecure())
		CheckError(err)

		// Obtain client stub
		cli := protos.NewConsensusServiceClient(connxn)

		client_objs[i] = cli

		clientDeadline := time.Now().Add(time.Duration(5) * time.Second)
		ctx, _ := context.WithDeadline(context.Background(), clientDeadline)

		// ReplicaReady is an RPC defined to inform the other replica about our connection
		_, err = cli.ReplicaReady(ctx, &empty.Empty{})
		CheckError(err)

		log.Printf("\nConnected to replica %v\n", i)

	}

	go node.RunElectionTimer()

	node.raft_node_mutex.Lock()
	log.Printf("\nLocked in ConnectToPeerReplicas\n")

	node.electionTimerRunning = true
	node.peer_replica_clients = client_objs

	log.Printf("\nUnLocked in ConnectedTOPeerReplicas\n")
	node.raft_node_mutex.Unlock()
}

// this goroutine will keep monitoring all connections and try to re-establish connections that die
// func (node *RaftNode) MonitorConnections() {

// 	for {

// 		for i := 0; i < node.n_replicas; i++ {

// 			if i == node.replica_id {
// 				continue
// 			}

// 			response, err := cli.ReplicaReady(context.Background(), &empty.Empty{})

// 			conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
// 			if err != nil {
// 				log.Fatalf("Did not connect: %v", err)
// 			}

// 		}

// 		time.Sleep(1 * time.Second)

// 	}

// }

// Apply committed entries to our state machine.
func (node *RaftNode) ApplyToStateMachine() {

	for {

		log.Printf("\nIn ApplyToStateMachine\n")
		to_commit := <-node.commits_ready
		log.Printf("\nReceived commit(s)\n")

		// Find which entries we have to apply.
		node.raft_node_mutex.Lock()
		log.Printf("\nLocked in ApplyToStateMachine\n")
		defer node.raft_node_mutex.Unlock()

		var entries []protos.LogEntry

		if node.commitIndex > node.lastApplied {
			entries = node.log[node.lastApplied+1 : node.lastApplied+to_commit]
			log.Printf("Operations to be applied to kv_store\n")
		} else {
			log.Printf("Fatal: node.commitIndex <= node.lastApplied in ApplyToStateMachine\n")
			return
		}

		for _, entry := range entries {
			formData := url.Values{
				"value": {entry.Operation[2]},
			}

			timeout := time.Duration(100 * time.Microsecond)
			client := http.Client{
				Timeout: timeout,
			}

			switch entry.Operation[0] {

			case "POST":
				formData := url.Values{
					"value": {entry.Operation[2]},
				}

				req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:%d/%s", node.kvstore_addr, entry.Operation[1]), bytes.NewBufferString(formData.Encode()))
				if err != nil {
					log.Printf("\nError in http.NewRequest: %v\n", err)
					return
				}

				req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")

				resp, err := client.Do(req)
				if err != nil {
					log.Printf("\nError in client.Do(req): %v\n", err)
					return
				}

				defer resp.Body.Close()
				break

			case "PUT":

				req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("http://localhost:%d/%s", node.kvstore_addr, entry.Operation[1]), bytes.NewBufferString(formData.Encode()))
				CheckError(err)
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")

				resp, err := client.Do(req)
				CheckError(err)
				defer resp.Body.Close()
				break

			case "DELETE":
				req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("http://localhost:%d/%s", node.kvstore_addr, entry.Operation[1]), bytes.NewBufferString(""))
				CheckError(err)

				resp, err := client.Do(req)
				CheckError(err)
				defer resp.Body.Close()

			case "NO-OP":
				log.Printf("\nNO-OP encountered\n")

			default:
				log.Printf("\nFatal: Invalid operation: %v\n", entry.Operation[0])

			}

		}
		node.lastApplied = node.lastApplied + to_commit
		log.Println("Required Operations done to kv_store; Current lastApplied: %v", node.lastApplied)
	}
}
