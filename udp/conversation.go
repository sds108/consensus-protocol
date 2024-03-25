package main // Declares that this file is part of the main package.

import (
	"fmt" // Import the fmt package for printing.
	"net"
	"sync"
	// Import the sync package for mutexes.
	// Import the time package for sleeps.
)

// Own Conversation ID ///////////////////////////////////////////////////////////////
var conversation_id_self = uint32(0)

// Structure container, connection free and instead dependent on the conversation's ID
type conversation struct {
	// Conversation ID
	conversation_id uint32

	// Buffer for incoming packets
	incoming map[uint32]Pckt

	// Mutex for the buffer with incoming packets
	incoming_lock sync.Mutex

	// Mutex for the receivedPackets map
	received_lock      sync.Mutex
	receivedPackets    map[uint32]bool
	highestSeqReceived uint32
}

// Conversation Constructor
func newConversation(conversation_id uint32) *conversation {
	return &conversation{
		conversation_id:    conversation_id,
		incoming:           make(map[uint32]Pckt),
		receivedPackets:    make(map[uint32]bool),
		highestSeqReceived: 0,
	}
}

// Handle Conversation
func (conv *conversation) ARQ_Receive(conn *net.UDPConn, addr *net.UDPAddr, pckt Pckt) {
	fmt.Printf("Received packet: ConversationId=%s, SequenceNumber=%d, Data=%s, IsFinal=%t\n", pckt.Header.ConvID, pckt.Header.SequenceNum, pckt.Body, pckt.Header.Final)

	// Lock Received Map
	conv.received_lock.Lock()
	defer conv.received_lock.Unlock()

	// Marks the packet as received
	conv.receivedPackets[pckt.Header.SequenceNum] = true

	// Updates the highest sequence number if required
	if pckt.Header.SequenceNum > conv.highestSeqReceived {
		conv.highestSeqReceived = pckt.Header.SequenceNum
	}

	// Mutex lock the incoming packets buffer
	conv.incoming_lock.Lock()

	// Append packet to incoming buffer
	//(conv.incoming)[pckt.Header.SequenceNum] = data

	// Unlock Mutex lock
	conv.incoming_lock.Unlock()

	// Makes a list of the missing packets
	var missingPackets []uint32
	for i := uint32(0); i <= conv.highestSeqReceived; i++ {
		if !(conv.receivedPackets)[i] {
			missingPackets = append(missingPackets, i)
		}
	}

	// Send an ACK for the received packet, including any missing packets.
	// ackPacket := Packet{
	// 	ConversationId: conv.conversation_id,
	// 	SequenceNumber: data.SequenceNumber,
	// 	Ack:            true,
	// 	MissingPackets: missingPackets,
	// }
	// ackData, _ := serializePacket(ackPacket)
	// if _, err := conn.WriteToUDP(ackData, addr); err != nil {
	// 	log.Printf("Failed to send ACK for packet %d: %v", data.SequenceNumber, err)
	// } else {
	// 	fmt.Printf("Sent ACK for packet: %d with missing packets: %v\n", ackPacket.SequenceNumber, ackPacket.MissingPackets)
	// }
}
