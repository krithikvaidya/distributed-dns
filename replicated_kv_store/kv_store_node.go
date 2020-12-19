package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"time"

	"github.com/krithikvaidya/distributed-dns/replicated_kv_store/protos"
	"google.golang.org/grpc"
)

var n_replica int

func init() {

	// Command line parameters
	flag.IntVar(&n_replica, "n", 5, "total number of replicas (default=5)")
	flag.Parse()

	log.SetFlags(0) // Turn off timestamps in log output.
	rand.Seed(time.Now().UnixNano())

}

func main() {

	fmt.Println("\nRaft-based Replicated Key Value Store\n")

	fmt.Printf("Enter the replica's id: ")
	var rid int32
	fmt.Scanf("%d", &rid)

	fmt.Printf("\nEnter the TCP network address that the replica should bind to (eg - :7890): ")
	var address string
	fmt.Scanf("%s", &address)

	tcpAddr, err := net.ResolveTCPAddr("tcp4", address)
	CheckError(err)

	listener, err := net.ListenTCP("tcp", tcpAddr)
	CheckError(err)

	fmt.Printf("\nSuccessfully bound to address %v\n", address)

	fmt.Printf("\nEnter the addresses of %v other replicas: \n", n_replica-1)

	rep_addrs := make([]string, n_replica)

	for i := int32(0); i < int32(n_replica); i++ {

		if i == rid {
			continue
		}

		fmt.Scanf("%s", &rep_addrs[i])

	}

	grpcServer := grpc.NewServer()

	// InitializeNode() is defined in raft_node.go
	node := InitializeNode(int32(n_replica), rid)

	// ConsensusService is defined in protos/replica.proto./
	// RegisterConsensusServiceServer is present in the generated .pb.go file
	protos.RegisterConsensusServiceServer(grpcServer, node)

	// gRPC Serve is blocking, so we do it on a separate goroutine
	go func() {

		err := grpcServer.Serve(listener)

		if err != nil {
			log.Printf("\nError in gRPC Serve: %v\n", err)
			os.Exit(1)
		}

	}()

	fmt.Printf("\ngRPC server listening...\n")

	fmt.Printf("\nPress enter when all other nodes are online.\n")
	var input rune
	fmt.Scanf("%c", &input)

	// Attempt to gRPC dial to other replicas. ConnectToPeerReplicas is defined in raft_node.go
	node.ConnectToPeerReplicas(rep_addrs)

	<-node.ready_chan // wait until all connections have been established.

	// dummy channel to ensure program doesn't exit. Remove it later
	all_connected := make(chan bool)
	<-all_connected

}
