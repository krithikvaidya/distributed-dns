package main

import (
	"context"
	"log"
	"net"
	"sync"
	"time"

	empty "github.com/golang/protobuf/ptypes/empty"
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
	n_replicas             int
	replica_id             int
	peer_replica_addresses []net.Conn
	raft_node_mutex        sync.Mutex
	node_state             RaftNodeState

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

func GetState(state RaftNodeState) string {

	switch state {

	case Follower:
		return "Follower"
	case Candidate:
		return "Candidate"
	case Leader:
		return "Leader"
	case Down:
		return "Down"

	}

	return "Invalid"

}

var replicas_ready int

func InitializeNode(n_replica int, rid int) *RaftNode {

	replicas_ready = 0

	rn := &RaftNode{

		n_replicas:             n_replica,
		replica_id:             rid,
		peer_replica_addresses: make([]net.Conn, n_replica),
		node_state:             Follower,

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

		defer connxn.Close()

		cli := protos.NewConsensusServiceClient(connxn)

		client_objs[i] = cli

		// Notify replica of connection
		_, err = client_objs[i].ReplicaReady(context.Background(), &empty.Empty{})

	}

}

func (node *RaftNode) ReplicaReady(ctx context.Context, in *empty.Empty) (*empty.Empty, error) {

	log.Printf("\nReceived ReplicaReady\n")
	return &empty.Empty{}, nil

}

func (node *RaftNode) RequestVote(ctx context.Context, in *protos.RequestVoteMessage) (*protos.RequestVoteResponse, error) {

	// ...
	return &protos.RequestVoteResponse{}, nil

}

func (node *RaftNode) AppendEntries(ctx context.Context, in *protos.AppendEntriesMessage) (*protos.AppendEntriesResponse, error) {

	// ...
	return &protos.AppendEntriesResponse{}, nil

}
