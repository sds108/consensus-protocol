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

	my_features = make([]uint16, 3)
	my_features[0] = 1
	my_features[1] = 0
	my_features[1] = 0
	loss_constant = 0
	defect_constant = 0.1
	duplicates_mode = 0
	debug_mode = false

	Startup()

	// Set up connection
	var err error
	conn, err = net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		log.Fatalf("Failed to dial UDP server: %v", err)
	}

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

	go Brainloop()
}
