// Boot up sequence for a server type node
package main

import (
	"log"
	"net"
	"sync"
)

func main() {
	// Check if this server already has a Conversation ID

	// Generate a Conversation ID for self if necessary
	conversation_id_self = generateConversationID()

	my_features = make([]uint16, 1)
	my_features[0] = 1
	loss_constant = 0
	i_am_server = true
	debug_mode = true

	// Resolve UDP Address to listen at
	addr, err := net.ResolveUDPAddr("udp", "0.0.0.0:"+SERVER_PORT_CONST)
	if err != nil {
		log.Fatal(err)
	}

	// Set up connection
	conn, err = net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("UDP server listening on port %s", SERVER_PORT_CONST)
	log.Println("My remote address is ", conn.RemoteAddr())
	log.Println("My local address is ", conn.LocalAddr())

	// Start Listener Thread
	globalWaitGroup := new(sync.WaitGroup)
	globalWaitGroup.Add(1)
	go listener()

	// Wait for waitgroup to finish
	globalWaitGroup.Wait()
}
