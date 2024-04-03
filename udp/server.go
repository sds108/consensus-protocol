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

	// Set up Server Port
	udpPort := "8080"
	i_am_server = true

	// Resolve UDP Address to listen at
	addr, err := net.ResolveUDPAddr("udp", "0.0.0.0:"+udpPort)
	if err != nil {
		log.Fatal(err)
	}

	// Set up connection
	conn, err = net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("UDP server listening on port %s", udpPort)
	log.Println("My name is ", conn.RemoteAddr())
	log.Println("My name is ", conn.LocalAddr())

	// Start Listener Thread
	globalWaitGroup := new(sync.WaitGroup)
	globalWaitGroup.Add(1)
	listener()

	// Wait for waitgroup to finish
	globalWaitGroup.Wait()
}
