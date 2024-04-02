// Boot up sequence for a client type node
package main

import (
	"log"
	"net"
	"sync"
	"time"
)

func main() {
	// Rule I am client
	i_am_server = false

	// Resolve UDP Server Address to contact
	serverAddr, err := net.ResolveUDPAddr("udp", "localhost:8080")
	if err != nil {
		log.Fatalf("Failed to resolve server address: %v", err)
	}

	// Set up connection
	conn, err = net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		log.Fatalf("Failed to dial UDP server: %v", err)
	}

	log.Println("My name is ", conn.RemoteAddr())
	log.Println("My name is ", conn.LocalAddr())

	// Start Listener Thread
	globalWaitGroup := new(sync.WaitGroup)
	globalWaitGroup.Add(1)
	go listener()
	go client_connect(serverAddr)

	// Wait for waitgroup to finish
	globalWaitGroup.Wait()
}

func client_connect(serverAddr *net.UDPAddr) {
	// Ping server for a Conversation ID if necessary
	pingPckt := Pckt{
		Header: PcktHeader{
			Magic:       MAGIC_CONST,
			Checksum:    0,
			ConvID:      conversation_id_self,
			PacketNum:   0,
			SequenceNum: 0,
			Type:        PING_REQ,
			IsFinal:     1,
		},
		Body: []byte{},
	}

	for conversation_id_self == 0 {
		// Send PING to server to obtain
		sendUDP(serverAddr, &pingPckt)

		time.Sleep(time.Second)
	}

	// Send SYN until made contact with server
	synPckt := Pckt{
		Header: PcktHeader{
			Magic:       MAGIC_CONST,
			Checksum:    0,
			ConvID:      conversation_id_self,
			PacketNum:   0,
			SequenceNum: 0,
			Type:        SYN,
			IsFinal:     1,
		},
		Body: []byte{},
	}

	for len(conversations) == 0 {
		// Send SYN to server to try make converstion
		sendUDP(serverAddr, &synPckt)

		time.Sleep(time.Second)
	}
}
