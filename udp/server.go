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
	i_am_server = true

	my_features = make([]uint16, 3)
	my_features[0] = 1 // Simple Math (boolean expression) Evaluation
	my_features[1] = 0 // SMT (Z3) Evaluation (not included in this project, planned for the future)
	my_features[2] = 0 // AI Image classification (not included in this project, planned for the future)

	loss_constant = 0.4 // 40% chance of a packet getting lost
	defect_constant = 0 // 0.0-1.0 (0-100%) chance of defecting to a vote
	duplicates_mode = 0 // 0-255 duplicates
	debug_mode = true   // debug mode prints everything

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
	go cleaner()

	// Wait for waitgroup to finish
	globalWaitGroup.Wait()
}
