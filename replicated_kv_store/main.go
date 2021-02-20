package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"
	"context"

	"github.com/gorilla/mux"
	"github.com/krithikvaidya/distributed-dns/replicated_kv_store/kv_store"
	"github.com/krithikvaidya/distributed-dns/replicated_kv_store/protos"
	"google.golang.org/grpc"
)

var n_replica int

// Start the local key-value store's HTTP server
func (node *RaftNode) StartKVStore(parent_ctx context.Context, addr string, num int) {
	// Creating a context for the key value store handler thread
	ctx, cancel := context.WithCancel(parent_ctx)
	defer cancel()

	filename := "600" + strconv.Itoa(num)

	kv := kv_store.InitializeStore(filename) // NewStore() defined in kv_store/restaccess_key_value.go

	r := mux.NewRouter()

	r.HandleFunc("/kvstore", kv.KvstoreHandler).Methods("GET")
	r.HandleFunc("/{key}", kv.PostHandler).Methods("POST")
	r.HandleFunc("/{key}", kv.GetHandler).Methods("GET")
	r.HandleFunc("/{key}", kv.PutHandler).Methods("PUT")
	r.HandleFunc("/{key}", kv.DeleteHandler).Methods("DELETE")

	// Create a server struct
	srv := &http.Server {
		Handler: r,
		Addr: addr,
	}

	kv_store_server := &HttpServer {
		srv: srv,
		close_chan: make(chan bool),
	}

	node.node_meta.kv_store_server = kv_store_server

	// Gracefully shut down the server if context is cancelled
	go func() {
		// Block till context is cancelled
		<-ctx.Done()

		// Shut down the server, on error, forcefully close the server
		if err_kv := node.node_meta.kv_store_server.srv.Shutdown(context.Background()); err_kv != nil {
			log.Printf("HTTP server Shutdown error: %v", err_kv)
			node.node_meta.kv_store_server.srv.Close()
		}

		node.node_meta.kv_store_server.close_chan<-true
	}()

	err := node.node_meta.kv_store_server.srv.ListenAndServe()

	// Handling code when the server is gracefully or forcefully shut down
	if err == http.ErrServerClosed {
		<-node.node_meta.kv_store_server.close_chan
	} else if err != nil {
		CheckErrorFatal(err)
	}
}

/*
 * External function to gracefully shut down the kv store server
 */
 func (node *RaftNode) StopKVStore() {
	// Stop the KV store service
	if err_kv := node.node_meta.kv_store_server.srv.Shutdown(context.Background()); err_kv != nil {
		log.Printf("HTTP server Shutdown error: %v", err_kv)
	}

	node.node_meta.kv_store_server.close_chan<-true
}

// Start a server to listen for client requests
func (node *RaftNode) StartRaftServer(parent_ctx context.Context, addr string) {
	// Creating a context for the raft server handler thread
	ctx, cancel := context.WithCancel(parent_ctx)
	defer cancel()

	node.node_meta.nodeAddress = addr //store address of the node

	r := mux.NewRouter()

	r.HandleFunc("/test", node.TestHandler).Methods("GET")
	r.HandleFunc("/{key}", node.PostHandler).Methods("POST")
	r.HandleFunc("/{key}", node.GetHandler).Methods("GET")
	r.HandleFunc("/{key}", node.PutHandler).Methods("PUT")
	r.HandleFunc("/{key}", node.DeleteHandler).Methods("DELETE")

	// Create a server struct
	srv := &http.Server {
		Handler: r,
		Addr:    addr,
	}

	raft_server := &HttpServer {
		srv: srv,
		close_chan: make(chan bool),
	}

	node.node_meta.raft_server = raft_server

	// Gracefully shut down the server if context is cancelled
	go func() {
		// Block till context is cancelled
		<-ctx.Done()

		// Shut down the server, on error, forcefully close the server
		if err_kv := node.node_meta.raft_server.srv.Shutdown(context.Background()); err_kv != nil {
			log.Printf("HTTP server Shutdown error: %v", err_kv)
			node.node_meta.raft_server.srv.Close()
		}

		node.node_meta.raft_server.close_chan<-true
	}()

	err := node.node_meta.raft_server.srv.ListenAndServe()

	if err == http.ErrServerClosed {
		<-node.node_meta.raft_server.close_chan
	} else if err != nil {
		CheckErrorFatal(err)
	}
}

/*
 * External function to gracefully shut down the raft server
 */
 func (node *RaftNode) StopRaftServer() {
	// Stop the raft store service
	if err_kv := node.node_meta.raft_server.srv.Shutdown(context.Background()); err_kv != nil {
		log.Printf("HTTP server Shutdown error: %v", err_kv)
	}

	node.node_meta.raft_server.close_chan<-true
}

/*
 * This function starts the gRPC server for the raft node and gracefully shuts it down when
 * context is cancelled.
 */
 func (node *RaftNode) StartGRPCServer(parent_ctx context.Context, grpc_address string, listener *net.TCPListener) {
	// Creating a context for the gRPC server thread
	ctx, cancel := context.WithCancel(parent_ctx)
	defer cancel()

	// Gracefully shut down the gRPC server if the context is cancelled
	go func() {
		// Block till the context is cancelled
		<-ctx.Done()

		// Stop the server
		node.node_meta.grpc_server.Stop()
	}()

	// Start the server
	log.Printf("\n Starting gRPC server at address %v...\n", grpc_address)
	err := node.node_meta.grpc_server.Serve(listener)
	CheckErrorFatal(err)
}

func init() {

	/*
	 * Workaround for a Go bug
	 * The Init() function for the testing package should be called
	 * before our init() function for parsing the command-line arguments
	 * of the `go test` command
	 */
	testing.Init()

	// Command line parameters
	flag.IntVar(&n_replica, "n", 5, "total number of replicas (default=5)")
	flag.Parse()

	log.SetFlags(0) // Turn off timestamps in log output.
	rand.Seed(time.Now().UnixNano())

}

/*
 * This function creates a raft node and imports the persistent
 * state information to the node.
 */
func setup_raft_node(id int, n_replicas int) *RaftNode {

	// Key value store address of the current node
	kv_addr := ":300" + strconv.Itoa(id)

	// Initialize node
	node := InitializeNode(int32(n_replicas), id, kv_addr)

	// Apply the persistent entries to the kv store
	go node.ApplyToStateMachine()

	return node
}

/*
 * This function connects an existing node to a raft system.
 *
 * It connects the current node to the other nodes. This mechanism includes the
 * initiation of their various services, like the
 * KV store server, the gRPC server and the Raft server.
 *
 * The `connect_chan` channel is used to signify the end of execution of this
 * function for synchronization and error handling.
 */
func (node *RaftNode) connect_raft_node(parent_ctx context.Context, id int, rep_addrs []string, testing bool, connect_chan chan bool) {
	// Creating a master context for the whole raft application
	ctx, _ := context.WithCancel(parent_ctx)
	node.node_meta.master_ctx, node.node_meta.master_cancel = context.WithCancel(ctx)

	// Starting KV store
	kvstore_addr := ":300" + strconv.Itoa(id)
	log.Println("Starting local key-value store...")
	go node.StartKVStore(ctx, kvstore_addr, id)

	/*
	 * Make a HTTP request to the test endpoint until a reply is obtained, indicating that
	 * the HTTP server is up
	 */

	test_addr := fmt.Sprintf("http://localhost%s/kvstore", kvstore_addr)

	if !testing {
		for {

			_, err := http.Get(test_addr)

			if err == nil {
				log.Printf("\nKey-value store up and listening at port %s\n", kvstore_addr)
				break
			}

		}
	}

	/*
	 * Connect the new node to the existing nodes.
	 * Attempt to gRPC dial to other replicas and obtain corresponding client stubs.
	 * ConnectToPeerReplicas is defined in raft_node.go.
	 */
	log.Println("Obtaining client stubs of gRPC servers running at peer replicas...")
	node.ConnectToPeerReplicas(ctx, rep_addrs)

	// Setting up and running the gRPC server
	grpc_address := ":500" + strconv.Itoa(id)

	tcpAddr, err := net.ResolveTCPAddr("tcp4", grpc_address)
	CheckErrorFatal(err)

	listener, err := net.ListenTCP("tcp", tcpAddr)
	CheckErrorFatal(err)

	node.node_meta.grpc_server = grpc.NewServer()

	/*
	 * ConsensusService is defined in protos/replica.proto./
	 * RegisterConsensusServiceServer is present in the generated .pb.go file
	 */
	protos.RegisterConsensusServiceServer(node.node_meta.grpc_server, node)

	// Running the gRPC server
	go node.StartGRPCServer(ctx, grpc_address, listener)

	// TODO: wait till grpc server up

	// Now we can start listening to client requests

	// Set up the server for the Raft functionalities
	server_address := ":400" + strconv.Itoa(id)
	log.Println("Starting raft replica server...")
	go node.StartRaftServer(ctx, server_address)

	test_addr = fmt.Sprintf("http://localhost%s/test", server_address)

	// Check whether the server is active
	if !testing {
		for {

			_, err = http.Get(test_addr)

			if err == nil {
				log.Printf("\nRaft replica server up and listening at port %s\n", server_address)
				break
			}

		}
	}

	connect_chan<-true
}

func main() {

	log.Println("Raft-based Replicated Key Value Store")

	log.Printf("Enter the replica's id: ")
	var rid int
	fmt.Scanf("%d", &rid)

	node := setup_raft_node(rid, n_replica)

	// Store the gRPC address of other replicas
	rep_addrs := make([]string, n_replica)
	for i := 0; i < n_replica; i++ {
		if i == rid {
			continue
		}

		rep_addrs[i] = ":500" + strconv.Itoa(i)
	}

	// Create context here
	ctx := context.Background()

	// Create a channel to make sure the connection is successful
	connect_chan := make(chan bool)

	// Connect the node
	node.connect_raft_node(ctx, rid, rep_addrs, false, connect_chan)

	<-connect_chan

	log.Printf("Node initialization successful")

	// dummy channel to ensure program doesn't exit. Remove it later
	all_connected := make(chan bool)
	<-all_connected

}
