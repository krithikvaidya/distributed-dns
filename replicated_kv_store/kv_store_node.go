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

	flag.IntVar(&n_replica, "n", 5, "total number of replicas (default=5)")
	flag.Parse()

	log.SetFlags(0) // Turn off timestamps in log output.
	rand.Seed(time.Now().UnixNano())

}

func main() {

	fmt.Println("\nRaft-based Replicated Key Value Store\n")

	fmt.Printf("Enter the replica's id: ")
	var rid int
	fmt.Scanf("%d", &rid)

	fmt.Printf("\nEnter the TCP network address that the replica should bind to (eg - :7890): ")
	var address string
	fmt.Scanf("%s", &address)

	tcpAddr, err := net.ResolveTCPAddr("tcp4", address)
	CheckError(err)

	// Start listening for TCP connections
	listener, err := net.ListenTCP("tcp", tcpAddr)
	CheckError(err)

	fmt.Printf("\nSuccessfully bound to address %v\n", address)

	fmt.Printf("\nEnter the addresses of %v other replicas: \n", n_replica-1)

	rep_addrs := make([]string, n_replica)

	for i := 0; i < n_replica; i++ {

		if i == rid {
			continue
		}

		fmt.Scanf("%s", &rep_addrs[i])

	}

	grpcServer := grpc.NewServer()

	node := InitializeNode(n_replica, rid)

	protos.RegisterConsensusServiceServer(grpcServer, node)

	go func() {

		err := grpcServer.Serve(listener)

		if err != nil {
			os.Exit(1)
		}

	}()

	fmt.Printf("\ngRPC server listening...\n")

	fmt.Printf("\nPress enter when all other nodes are online.\n")
	var input rune
	fmt.Scanf("%c", &input)

	node.ConnectToPeerReplicas(rep_addrs)
	c := make(chan bool)
	<-c

}
