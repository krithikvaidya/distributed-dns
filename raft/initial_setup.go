package raft

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"github.com/krithikvaidya/distributed-dns/raft/kv_store"
	"github.com/krithikvaidya/distributed-dns/raft/protos"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

// Start the local key-value store and the HTTP server it listens for requests on.
func (node *RaftNode) StartKVStore(ctx context.Context, addr string, num int, testing bool) {

	filename := "600" + strconv.Itoa(num)

	// InitializeStore is defined in kv_store/restaccess_key_value.go
	kv := kv_store.InitializeStore(filename)

	r := mux.NewRouter()

	r.HandleFunc("/kvstore", kv.KvstoreHandler).Methods("GET")
	r.HandleFunc("/{key}", kv.PostHandler).Methods("POST")
	r.HandleFunc("/{key}", kv.GetHandler).Methods("GET")
	r.HandleFunc("/{key}", kv.PutHandler).Methods("PUT")
	r.HandleFunc("/{key}", kv.DeleteHandler).Methods("DELETE")

	// Create a server struct
	srv := &http.Server{
		Handler: r,
		Addr:    addr,
	}

	srv.SetKeepAlivesEnabled(false)

	node.Meta.kv_store_server = srv

	// Gracefully shut down the server if context is cancelled
	go func() {

		// Block till context is cancelled
		<-ctx.Done()

		// Shut down the server. On error, forcefully close the server
		// We use HTTP's inbuilt Shutdown() and Close() methods for this.
		ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)

		if err_kv := node.Meta.kv_store_server.Shutdown(ctx); err_kv != nil {

			log.Printf("HTTP server Shutdown error: %v\n", err_kv)
			node.Meta.kv_store_server.Close()

		}

	}()

	err := node.Meta.kv_store_server.ListenAndServe()

	// Handling code for when the server is unexpectedly closed.
	if (err != nil) && (err != http.ErrServerClosed) {
		CheckErrorFatal(err)
	}

	if !testing {
		node.Meta.shutdown_chan <- "KV store shutdown successful."
	}

}

// HTTP server to listen for client requests
func (node *RaftNode) StartRaftServer(ctx context.Context, addr string, testing bool) {

	node.Meta.nodeAddress = addr // store address of the node

	r := mux.NewRouter()

	r.HandleFunc("/test", node.TestHandler).Methods("GET")
	r.HandleFunc("/{key}", node.PostHandler).Methods("POST")
	r.HandleFunc("/{key}", node.GetHandler).Methods("GET")
	r.HandleFunc("/{key}", node.PutHandler).Methods("PUT")
	r.HandleFunc("/{key}", node.DeleteHandler).Methods("DELETE")

	// Create a server struct
	raft_server := &http.Server{
		Handler: r,
		Addr:    addr,
	}

	raft_server.SetKeepAlivesEnabled(false)

	node.Meta.raft_server = raft_server

	// Gracefully shut down the server if context is cancelled
	go func() {

		// Block till context is cancelled
		<-ctx.Done()

		// Shut down the server. On error, forcefully close the server
		// We use HTTP's inbuilt Shutdown() and Close() methods for this.
		ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)

		if err_kv := node.Meta.raft_server.Shutdown(ctx); err_kv != nil {
			log.Printf("HTTP server Shutdown error: %v\n", err_kv)
			node.Meta.raft_server.Close()
		}

	}()

	err := node.Meta.raft_server.ListenAndServe()

	// Handling code for when the server is unexpectedly closed.
	if (err != nil) && (err != http.ErrServerClosed) {
		CheckErrorFatal(err)
	}

	if !testing {
		node.Meta.shutdown_chan <- "Client request listener shutdown successful."
	}

}

/*
This function starts the gRPC server for the raft node and shuts it down when
context is cancelled.
*/
func (node *RaftNode) StartGRPCServer(ctx context.Context, grpc_address string, listener *net.TCPListener, testing bool) {

	// Shut down the gRPC server if the context is cancelled
	go func() {

		// Block till the context is cancelled
		<-ctx.Done()

		// Stop the server
		node.Meta.grpc_server.GracefulStop()

		if !testing {
			node.Meta.shutdown_chan <- "gRPC server shutdown successful."
		}

	}()

	// Start the server
	log.Printf("\nStarting gRPC server at address %v...\n", grpc_address)
	err := node.Meta.grpc_server.Serve(listener) // Serve will return a non-nil error unless Stop or GracefulStop is called.

	CheckErrorFatal(err)
}

/*
This function initializes the node and imports the persistent
state information to the node.
*/
func Setup_raft_node(ctx context.Context, id int, n_replicas int, testing bool) *RaftNode {

	// Key value store address of the current node
	kv_addr := ":300" + strconv.Itoa(id)

	// InitializeNode is defined in raft_node.go
	node := InitializeNode(int32(n_replicas), id, kv_addr)

	// ApplyToStateMachine() is defined in raft_node.go
	go node.ApplyToStateMachine(ctx, testing)

	// Starting KV store
	log.Println("Starting local key-value store...")
	go node.StartKVStore(ctx, kv_addr, id, testing)

	/*
	 * Make a HTTP request to the test endpoint until a reply is obtained, indicating that
	 * the HTTP server is up
	 */
	test_addr := fmt.Sprintf("http://localhost%s/kvstore", kv_addr)

	for {

		_, err := http.Get(test_addr)

		if err == nil {
			log.Printf("\nKey-value store up and listening at port %s\n", kv_addr)
			break
		}

	}

	return node
}

/*
It connects the current node to the other nodes. This mechanism includes the
initiation of their various services, like the
KV store server, the gRPC server and the Raft server.

The `connect_chan` channel is used to signify the end of execution of this
function for synchronization and error handling.
*/
func (node *RaftNode) Connect_raft_node(ctx context.Context, id int, rep_addrs []string, testing bool) {

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

	node.Meta.grpc_server = grpc.NewServer()

	/*
	 * ConsensusService is defined in protos/replica.proto
	 * RegisterConsensusServiceServer is present in the generated .pb.go file
	 */
	protos.RegisterConsensusServiceServer(node.Meta.grpc_server, node)

	// Running the gRPC server
	go node.StartGRPCServer(ctx, grpc_address, listener, testing)

	// wait till grpc server is up
	connxn, err := grpc.Dial(grpc_address, grpc.WithInsecure())

	// below block may not be needed
	for err != nil {
		connxn, err = grpc.Dial(grpc_address, grpc.WithInsecure())
	}

	for {

		if connxn.GetState() == connectivity.Ready {
			break
		}

		time.Sleep(20 * time.Millisecond)

	}

	// Now we can start listening to client requests

	// Set up the server that listens for client requests.
	server_address := ":400" + strconv.Itoa(id)
	log.Println("Starting raft replica server...")
	go node.StartRaftServer(ctx, server_address, testing)

	test_addr := fmt.Sprintf("http://localhost%s/test", server_address)

	// Check whether the server is active
	for {

		_, err = http.Get(test_addr)

		if err == nil {
			log.Printf("\nRaft replica server up and listening at port %s\n", server_address)
			break
		}

	}

}
