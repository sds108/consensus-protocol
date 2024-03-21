package main // Declares that this file is part of the main package.

import (
	"fmt" // Import the fmt package for printing.
	"log"
	"net"
	"sync" // Import the sync package for mutexes.
	// Import the time package for sleeps.
)

// Structure container, connection free and instead dependent on the conversation's ID
type conversation struct {
	// Conversation ID
	conversation_id string

	// Buffer for incoming packets
	incoming map[int]Packet

	// Mutex for the buffer with incoming packets
	incoming_lock sync.Mutex

	// Mutex for the receivedPackets map
	received_lock      sync.Mutex
	receivedPackets    map[int]bool
	highestSeqReceived int
}

// Conversation Constructor
func newConversation(conversation_id string) *conversation {
	return &conversation{
		conversation_id:    conversation_id,
		incoming:           make(map[int]Packet),
		receivedPackets:    make(map[int]bool),
		highestSeqReceived: 0,
	}
}

// Receive Method
func (conv *conversation) Receive(packet Packet) {
	// Mutex lock the incoming packets buffer
	//conv.incoming_lock.Lock()

	// Append packet to incoming buffer
	//conv.incoming = append(conv.incoming, packet)

	// Unlock Mutex lock
	//conv.incoming_lock.Unlock()
}

// Handle Conversation
func (conv *conversation) ARQ_Receive(conn *net.UDPConn, addr *net.UDPAddr, data Packet) {
	fmt.Printf("Received packet: ConversationId=%s, SequenceNumber=%d, Data=%s, IsFinal=%t\n", data.ConversationId, data.SequenceNumber, data.Data, data.IsFinal)

	// Lock Received Map
	conv.received_lock.Lock()
	defer conv.received_lock.Unlock()

	// Marks the packet as received
	conv.receivedPackets[data.SequenceNumber] = true

	// Updates the highest sequence number if required
	if data.SequenceNumber > conv.highestSeqReceived {
		conv.highestSeqReceived = data.SequenceNumber
	}

	// Mutex lock the incoming packets buffer
	conv.incoming_lock.Lock()

	// Append packet to incoming buffer
	(conv.incoming)[data.SequenceNumber] = data

	// Unlock Mutex lock
	conv.incoming_lock.Unlock()

	// Makes a list of the missing packets
	var missingPackets []int
	for i := 0; i <= conv.highestSeqReceived; i++ {
		if !(conv.receivedPackets)[i] {
			missingPackets = append(missingPackets, i)
		}
	}

	// Send an ACK for the received packet, including any missing packets.
	ackPacket := Packet{
		ConversationId: conv.conversation_id,
		SequenceNumber: data.SequenceNumber,
		Ack:            true,
		MissingPackets: missingPackets,
	}
	ackData, _ := serializePacket(ackPacket)
	if _, err := conn.WriteToUDP(ackData, addr); err != nil {
		log.Printf("Failed to send ACK for packet %d: %v", data.SequenceNumber, err)
	} else {
		fmt.Printf("Sent ACK for packet: %d with missing packets: %v\n", ackPacket.SequenceNumber, ackPacket.MissingPackets)
	}
}
