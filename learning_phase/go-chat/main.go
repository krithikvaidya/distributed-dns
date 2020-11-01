package main

/**

Protocol for sending/receiving messages:
- Each message will be of length 256 bytes.
- There will be 6 types of messages:

1. authenticate~<server_password>~<client_username>~\n  --> sent from client to server
2. authenticated~\n  --> sent from server to client
3. pm~<username>~<message>~\n  --> from client to server, unicast message
4. broadcast~<message>~\n  --> from client to server, broadcast message
5. message~<username>~<type>~<message>~\n  --> from server to client(s). Type indicates unicast/broadcast.
6. terminate~<reason>~\n  --> from server to client, for graceful shutdown

Possible extensions:
dockerize the application
use gorilla websockets instead of raw sockets, get a web-based frontend
use RPCs for communication instead of raw sockets
persist messages
write tests
add security
deploy to heroku

*/

import (
	
)

var (
	// Define variables for command line parameters here if using init() function
	/**
	Ideally, we would have command line parameters for the following purposes:
	- For starting the server process
	- For setting the server properties like IP address, port number and password
	- For setting the parameters with which the client connects to the server like the
	  username, target server IP address, port number and the password.
	*/
)



func main() {

	/**
	- Create a cancellable context to be passed to client/server functions
	- Listen for appropriate termination signals (refer to https://gobyexample.com/signals) in a separate goroutine
	- The above goroutine should call the context's cancel function on receiving a termination signal,
	  and wait for the client/server function called below to complete its execution (how will this goroutine know
	  when the client/server function called has completed execution?). Then it can finally terminate the program.
	- The main function should then decide whether the execution will be in client mode or in server mode
	  (by reading the command line parameters). On deciding, it should create & initialize a new struct of
	  client/server type (which will be defined in client.go and server.go), and call the respective Run() function  
	*/

}

