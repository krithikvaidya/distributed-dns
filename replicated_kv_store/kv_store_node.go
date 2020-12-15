package main

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"time"

	"google.golang.org/grpc"
)

func main() {

	log.SetFlags(0) // Turn off timestamps in log output.
	rand.Seed(time.Now().UnixNano())

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

	grpcServer := grpc.NewServer()

	err = grpcServer.Serve(listener)
	checkError(err)

}
