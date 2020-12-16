package main

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/krithikvaidya/distributed-dns/replicated_kv_store/protos"
	"google.golang.org/grpc"
)

type RaftNodeState int

const (
	Follower RaftNodeState = iota
	Candidate
	Leader
	Down
)

type RaftNode struct {
	protos.UnimplementedConsensusServiceServer
	ready_chan           chan bool
	n_replicas           int
	replicas_ready       int
	replica_id           int
	peer_replica_clients []protos.ConsensusServiceClient
	raft_node_mutex      sync.Mutex
	node_state           RaftNodeState

	// States mentioned in figure 2 of the paper:

	// State to be maintained on all replicas (TODO: persist)
	currentTerm int
	votedFor    int
	log         []int

	// State to be maintained on all replicas
	commitIndex        int
	lastApplied        int
	state              RaftNodeState
	electionResetEvent time.Time

	// State to be maintained on the leader
	nextIndex  []int
	matchIndex []int
}

func InitializeNode(n_replica int, rid int) *RaftNode {

	rn := &RaftNode{

		n_replicas:           n_replica,
		ready_chan:           make(chan bool),
		replicas_ready:       0,
		replica_id:           rid,
		peer_replica_clients: make([]protos.ConsensusServiceClient, n_replica),
		node_state:           Follower,

		currentTerm: 0, // unpersisted
		votedFor:    -1,
		log:         make([]int, 10000),

		commitIndex: 0,
		lastApplied: 0,
	}

	return rn

}

func (node *RaftNode) ConnectToPeerReplicas(rep_addrs []string) {

	client_objs := make([]protos.ConsensusServiceClient, node.n_replicas)

	for i := 0; i < node.n_replicas; i++ {

		if i == node.replica_id {
			continue
		}

		connxn, err := grpc.Dial(rep_addrs[i], grpc.WithInsecure())
		CheckError(err)

		cli := protos.NewConsensusServiceClient(connxn)

		client_objs[i] = cli

		response, err := cli.ReplicaReady(context.Background(), &empty.Empty{})
		CheckError(err)

	}

	node.peer_replica_clients = client_objs

}

func (node *RaftNode) ReplicaReady(ctx context.Context, in *empty.Empty) (*empty.Empty, error) {

	node.raft_node_mutex.Lock() // Multiple instances of ReplicaReady method may run parallely

	log.Printf("\nReceived ReplicaReady Notification from %v\n", in.Id)
	node.replicas_ready += 1

	if node.replicas_ready == node.n_replicas-1 {

		node.raft_node_mutex.Unlock()

		// Using defer does not work here
		go func(node *RaftNode) { node.ready_chan <- true }(node)

		log.Printf("\nAll replicas have connected.\n")

	} else {
		node.raft_node_mutex.Unlock()
	}

	return &empty.Empty{}, nil
}
