package main // Declares that this file is part of the main package.

import (
	"fmt" // Import the fmt package for printing.
	"log"
	"net"
	"sync" // Import the sync package for mutexes.
	// Import the time package for sleeps? might not need!!
)

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

// newConversation creates a new conversation instance
func newConversation(conversation_id uint32) *conversation {
	return &conversation{
		conversation_id:    conversation_id,
		incoming:           make(map[uint32]Pckt),
		receivedPackets:    make(map[uint32]bool),
		highestSeqReceived: 0,
	}
}

// ARQ_Receive handles incoming packets, checks for duplicates, and sends ACKs/NAKs
func (conv *conversation) ARQ_Receive(conn *net.UDPConn, addr *net.UDPAddr, pckt Pckt) {
	fmt.Printf("Received packet: ConversationId=%d, SequenceNumber=%d, Data=%s, IsFinal=%t\n", pckt.Header.ConvID, pckt.Header.SequenceNum, string(pckt.Body), pckt.Header.IsFinal != 0)

	// Lock Received Map
	conv.received_lock.Lock()
	defer conv.received_lock.Unlock()

	// Check for duplicate packets
	if _, exists := conv.receivedPackets[pckt.Header.SequenceNum]; exists {
		log.Printf("Duplicate packet received: %d. Sending ACK.", pckt.Header.SequenceNum)
		// Send ACK for duplicate packet
		conv.sendACK(conn, addr, pckt.Header.SequenceNum)
		return
	}

	// Marks the packet as received
	conv.receivedPackets[pckt.Header.SequenceNum] = true

	// Updates the highest sequence number if required
	if pckt.Header.SequenceNum > conv.highestSeqReceived {
		conv.highestSeqReceived = pckt.Header.SequenceNum
	}

	// Mutex lock the incoming packets buffer
	conv.incoming_lock.Lock()

	// Append packet to incoming buffer
	(conv.incoming)[pckt.Header.SequenceNum] = pckt

	// Unlock Mutex lock
	conv.incoming_lock.Unlock()

	// Makes a list of the missing packets
	missingPackets := []uint32{}
	for seq := uint32(1); seq <= conv.highestSeqReceived; seq++ {
		if _, ok := conv.receivedPackets[uint32(seq)]; !ok {
			missingPackets = append(missingPackets, seq)
			conv.sendNAK(conn, addr, uint32(seq))
		}
	}
	missingPacketsUint32 := make([]uint32, len(missingPackets))
	for i, v := range missingPackets {
		missingPacketsUint32[i] = uint32(v)
	}

	// Send an ACK for the received packet, including any missing packets.
	ackPacket := Pckt{
		Header: PcktHeader{
			ConvID:      conv.conversation_id,
			SequenceNum: pckt.Header.SequenceNum,
			Type:        ACK,
		},
		Body:           []byte{},
		MissingPackets: missingPacketsUint32,
	}

	ackData, _ := SerializePacket(ackPacket) // Updated to use SerializePacket
	if _, err := conn.WriteToUDP(ackData, addr); err != nil {
		log.Printf("Failed to send ACK for packet %d: %v", pckt.Header.SequenceNum, err)
	} else {
		fmt.Printf("Sent ACK for packet: %d with missing packets: %v\n", ackPacket.Header.SequenceNum, ackPacket.MissingPackets)
	}
}

// sendNAK sends a NAK for a missing packet
func (conv *conversation) sendNAK(conn *net.UDPConn, addr *net.UDPAddr, missingSeqNum uint32) {
	nakPacket := Pckt{
		Header: PcktHeader{
			ConvID:      conv.conversation_id,
			SequenceNum: missingSeqNum,
			Type:        NAK, // Assuming you have a NAK constant defined
		},
		Body: []byte{},
	}
	nakData, _ := SerializePacket(nakPacket)
	if _, err := conn.WriteToUDP(nakData, addr); err != nil {
		log.Printf("Failed to send NAK for packet %d: %v", missingSeqNum, err)
	} else {
		log.Printf("Sent NAK for missing packet: %d\n", missingSeqNum)
	}
}

// sendACK sends an ACK for a received packet
func (conv *conversation) sendACK(conn *net.UDPConn, addr *net.UDPAddr, seqNum uint32) {
	ackPacket := Pckt{
		Header: PcktHeader{
			ConvID:      conv.conversation_id,
			SequenceNum: seqNum,
			Type:        ACK,
		},
		Body: []byte{},
	}
	ackData, _ := SerializePacket(ackPacket)
	if _, err := conn.WriteToUDP(ackData, addr); err != nil {
		log.Printf("Failed to send ACK for packet %d: %v", seqNum, err)
	}
}
