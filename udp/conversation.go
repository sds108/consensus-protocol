package main // Declares that this file is part of the main package.

import (
	"fmt" // Import the fmt package for printing.
	"log"
	"math/rand"
	"net"
	"sync" // Import the sync package for mutexes.
	"time"
)

// Define PacketState struct
type PacketState struct {
	AckReceived bool      // Indicates if ACK has been received for the packet
	LastSent    time.Time // The last time the packet was sent
}

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

	// Additions for Selective Repeat ARQ
	packetStates      map[uint32]*PacketState // For keeping track of packet's state
	packetStatesMutex sync.Mutex              // Mutex for packetStates map
	windowStart       uint32
	windowSize        uint32
	totalPackets      uint32 // Total number of packets expected to be sent
}

// newConversation creates a new conversation instance
func newConversation(conversation_id uint32) *conversation {
	return &conversation{
		conversation_id:    conversation_id,
		incoming:           make(map[uint32]Pckt),
		receivedPackets:    make(map[uint32]bool),
		highestSeqReceived: 0,
		packetStates:       make(map[uint32]*PacketState),
		windowStart:        1,
		windowSize:         10, // Example window size
	}
}

// ARQ_Receive handles incoming packets, checks for duplicates, and sends ACKs/NAKs, updates receivedPackets
func (conv *conversation) ARQ_Receive(conn *net.UDPConn, addr *net.UDPAddr, pckt Pckt) {
	log.Printf("Received packet: ConversationId=%d, SequenceNumber=%d, Data=%s, IsFinal=%t\n", pckt.Header.ConvID, pckt.Header.SequenceNum, string(pckt.Body), pckt.Header.IsFinal != 0)

	// Lock Received Map
	conv.received_lock.Lock()
	defer conv.received_lock.Unlock()

	// Check for duplicate packets
	if _, exists := conv.receivedPackets[pckt.Header.SequenceNum]; exists {
		log.Printf("Duplicate packet received: %d. Sending ACK again.", pckt.Header.SequenceNum)
		conv.sendACK(conn, addr, pckt.Header.SequenceNum)
		return // Return early to avoid any window movement or state change
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

	for seq := uint32(1); seq <= conv.highestSeqReceived; seq++ {
		if _, ok := conv.receivedPackets[uint32(seq)]; !ok {
			log.Printf("Missing packet: %d. Sending NAK.", seq)
			conv.sendNAK(conn, addr, uint32(seq))
		}
	}

	// Send an ACK for the received packet
	conv.sendACK(conn, addr, pckt.Header.SequenceNum)
}

// sendNAK sends a NAK for a missing packet
func (conv *conversation) sendNAK(conn *net.UDPConn, addr *net.UDPAddr, missingSeqNum uint32) {
	nakPacket := Pckt{
		Header: PcktHeader{
			ConvID:      conv.conversation_id,
			SequenceNum: missingSeqNum,
			Type:        NAK,
		},
		Body: []byte{},
	}
	nakData, err := SerializePacket(nakPacket)
	if err != nil {
		log.Printf("Error serializing NAK for packet %d: %v", missingSeqNum, err)
		return
	}
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
	ackData, err := SerializePacket(ackPacket)
	if err != nil {
		log.Printf("Error serializing ACK for packet %d: %v", seqNum, err)
		return
	}
	if _, err := conn.WriteToUDP(ackData, addr); err != nil {
		log.Printf("Failed to send ACK for packet %d: %v", seqNum, err)
	}
}

// Sends a packet and updates its state in the packetStates map, marks it as sent and records the sending time.
func (conv *conversation) sendPacket(conn *net.UDPConn, addr *net.UDPAddr, seqNum uint32, data []byte, packetType uint16) error {
	packet := Pckt{
		Header: PcktHeader{
			ConvID:      conv.conversation_id,
			SequenceNum: seqNum,
			Type:        packetType, // Use appropriate constant for DATA, ACK, or NAK
			IsFinal:     uint16(0),  // Initialize as 0
		},
		Body: data,
	}

	if seqNum == conv.totalPackets {
		packet.Header.IsFinal = 1 // Set to 1 if it's the final packet
	}

	packetData, err := SerializePacket(packet)
	if err != nil {
		log.Printf("Error serializing packet: %v", err)
		return err
	}

	if _, err := conn.Write(packetData); err != nil {
		log.Printf("Error sending packet: %v", err)
		return err
	}

	log.Printf("Packet sent: ConversationId=%d, SequenceNumber=%d, Type=%d, IsFinal=%t\n", conv.conversation_id, seqNum, packetType, uint16ToBool(packet.Header.IsFinal))

	conv.packetStatesMutex.Lock()
	if state, exists := conv.packetStates[seqNum]; exists {
		state.LastSent = time.Now() // Update LastSent after successful send
	} else {
		// If for some reason the packet state doesn't exist, create it
		conv.packetStates[seqNum] = &PacketState{AckReceived: false, LastSent: time.Now()}
	}
	conv.packetStatesMutex.Unlock()

	return nil
}

// Manages the sending window by sending packets within the window size, ensures packets are sent in sequence
// and handles the window sliding as packets are acknowledged.
func (conv *conversation) sendWindowPackets(conn *net.UDPConn, addr *net.UDPAddr) {
	conv.incoming_lock.Lock()
	defer conv.incoming_lock.Unlock()

	for seq := conv.windowStart; seq < conv.windowStart+conv.windowSize; seq++ {
		if packetState, exists := conv.packetStates[seq]; !exists || !packetState.AckReceived {
			data := []byte{} // Your data here
			conv.sendPacket(conn, addr, seq, data, DATA)
			conv.packetStates[seq] = &PacketState{LastSent: time.Now()}
		}
	}
}

func (conv *conversation) handleACK(seqNum uint32) {
	conv.packetStatesMutex.Lock() // Ensure thread-safe access to packetStates
	defer conv.packetStatesMutex.Unlock()

	if packetState, exists := conv.packetStates[seqNum]; exists {
		packetState.AckReceived = true
		// Move window if this ACK is for the first packet in the window
		if seqNum == conv.windowStart {
			for ; conv.packetStates[conv.windowStart] != nil && conv.packetStates[conv.windowStart].AckReceived; conv.windowStart++ {
				// Optionally, remove acknowledged packet state to save memory
				delete(conv.packetStates, conv.windowStart)
			}
		}
	}
	log.Printf("Window moved to start at %d", conv.windowStart)
}

// Immediately resends a packet when a NAK is received for it
func (conv *conversation) handleNAK(seqNum uint32, conn *net.UDPConn, addr *net.UDPAddr) {
	conv.packetStatesMutex.Lock()
	defer conv.packetStatesMutex.Unlock()

	// Check if the requested sequence number is within the current sending window
	if seqNum >= conv.windowStart && seqNum < conv.windowStart+conv.windowSize {
		// The packet is within the window, proceed with resend
		data := []byte{} // Ideally, retrieve the original data if stored
		if err := conv.sendPacket(conn, addr, seqNum, data, DATA); err == nil {
			if state, exists := conv.packetStates[seqNum]; exists {
				state.LastSent = time.Now() // Update LastSent after successful resend
			} else {
				// If for some reason the packet state doesn't exist, create it
				conv.packetStates[seqNum] = &PacketState{AckReceived: false, LastSent: time.Now()}
			}
		}
	} else {
		// The packet is outside the window, log this event for debugging purposes
		log.Printf("Received NAK for packet %d which is outside the current window [%d, %d). Ignoring.", seqNum, conv.windowStart, conv.windowStart+conv.windowSize)
	}
}

// Uses a ticker to periodically check for packets that have not been acknowledged within a certain timeout and resends them.
func (conv *conversation) checkForRetransmissions(conn *net.UDPConn, addr *net.UDPAddr) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		conv.packetStatesMutex.Lock() // Ensure thread-safe access to packetStates
		for seq, state := range conv.packetStates {
			if !state.AckReceived && time.Since(state.LastSent) > time.Second*5 { // Example timeout
				// Resend packet
				data := []byte{} // Your data here
				if err := conv.sendPacket(conn, addr, seq, data, DATA); err == nil {
					state.LastSent = time.Now() // Update LastSent after successful resend
				}
			}
		}
		conv.packetStatesMutex.Unlock()
	}
}

// Implementation of timers for managing packet resends, periodically checks the packetStates map and
// resends any packets that have not been acknowledged after a certain time interval.
func (conv *conversation) managePacketResends(conn *net.UDPConn, addr *net.UDPAddr) {
	ticker := time.NewTicker(1 * time.Second) // Adjust the ticker interval as needed
	defer ticker.Stop()

	for range ticker.C {
		conv.packetStatesMutex.Lock()
		for seqNum, packetState := range conv.packetStates {
			if !packetState.AckReceived && time.Since(packetState.LastSent) > 5*time.Second { // Example resend interval
				log.Printf("Resend interval expired for packet %d, resending...\n", seqNum)
				// Assuming you have a method to construct and serialize a packet for resending
				data := []byte{} // Your data here, or retrieve the original data if stored
				if err := conv.sendPacket(conn, addr, seqNum, data, DATA); err == nil {
					packetState.LastSent = time.Now() // Update LastSent after successful resend
				}
			}
		}
		conv.packetStatesMutex.Unlock()
	}
}

// Adapted version of sendPacketsWithoutLoss as a method of the conversation struct
func (conv *conversation) sendPacketsWithoutLoss(conn *net.UDPConn, addr *net.UDPAddr, totalPackets int) {
	for i := 1; i <= totalPackets; i++ {
		packet := Pckt{
			Header: PcktHeader{
				ConvID:      conv.conversation_id, // Use the struct's conversation_id
				SequenceNum: uint32(i),
				Type:        DATA,      // Assuming DATA is a defined constant for packet type
				IsFinal:     uint16(0), // Initialize as 0
			},
			Body: []byte(fmt.Sprintf("Packet without loss %d", i)),
		}

		if i == totalPackets {
			packet.Header.IsFinal = 1 // Set to 1 if it's the final packet
		}

		packetData, err := SerializePacket(packet)
		if err != nil {
			log.Printf("Error serializing packet: %v", err)
			continue
		}

		if _, err := conn.Write(packetData); err != nil {
			log.Printf("Error sending packet: %v", err)
			continue
		}

		log.Printf("Packet sent without loss: SequenceNumber=%d", i)
	}
}

// Simulates packet loss by randomly dropping packets based on a loss rate.
func (conv *conversation) sendPacketsWithSimulatedLoss(conn *net.UDPConn, addr *net.UDPAddr, totalPackets int, lossRate float64) {
	rand.Seed(time.Now().UnixNano()) // Seed the random number generator

	for i := 1; i <= totalPackets; i++ {
		if rand.Float64() > lossRate {
			// Only send the packet if the random number is greater than the loss rate
			packet := Pckt{
				Header: PcktHeader{
					ConvID:      conv.conversation_id,
					SequenceNum: uint32(i),
					Type:        DATA,
					IsFinal:     uint16(0), // Initialize as 0
				},
				Body: []byte(fmt.Sprintf("Packet with simulated loss %d", i)),
			}

			if i == totalPackets {
				packet.Header.IsFinal = 1 // Set to 1 if it's the final packet
			}

			packetData, err := SerializePacket(packet)
			if err != nil {
				log.Printf("Error serializing packet: %v", err)
				continue
			}

			if _, err := conn.Write(packetData); err != nil {
				log.Printf("Error sending packet: %v", err)
				continue
			}

			log.Printf("Packet sent with simulated loss: SequenceNumber=%d", i)
		} else {
			log.Printf("Simulating loss for packet: SequenceNumber=%d", i)
		}
	}
}

// Sends a packet and immediately sends a duplicate of it.
func (conv *conversation) sendAndDuplicatePacket(conn *net.UDPConn, addr *net.UDPAddr, seqNum int) {
	packet := Pckt{
		Header: PcktHeader{
			ConvID:      conv.conversation_id, // Use the struct's conversation_id
			SequenceNum: uint32(seqNum),
			Type:        DATA,      // Assuming DATA is a defined constant for packet type
			IsFinal:     uint16(0), // Initialize as 0
		},
		Body: []byte(fmt.Sprintf("Test duplicate, packet %d", seqNum)),
	}

	if seqNum == int(conv.totalPackets) {
		packet.Header.IsFinal = 1 // Set to 1 if it's the final packet
	}

	packetData, err := SerializePacket(packet)
	if err != nil {
		log.Printf("Error serializing packet: %v", err)
		return
	}

	// Send the packet
	if _, err := conn.Write(packetData); err != nil {
		log.Printf("Error sending packet: %v", err)
		return
	}
	log.Printf("Packet sent: SequenceNumber=%d", seqNum)

	// Immediately send the duplicate
	if _, err := conn.Write(packetData); err != nil {
		log.Printf("Error sending duplicate packet: %v", err)
		return
	}
	log.Printf("Duplicate packet sent: SequenceNumber=%d", seqNum)
}

// Tests the ARQ mechanism by sending a specified number of packets and handling the retransmission of any lost packets.
func (conv *conversation) testARQMechanism(conn *net.UDPConn, addr *net.UDPAddr, totalPackets int) {
	log.Printf("Starting ARQ mechanism test with %d total packets.", totalPackets)

	for i := 1; i <= totalPackets; i++ {
		log.Printf("Constructing packet %d of %d.", i, totalPackets)
		// Construct and send packet
		packet := Pckt{
			Header: PcktHeader{
				ConvID:      conv.conversation_id,
				SequenceNum: uint32(i),
				Type:        DATA,
				IsFinal:     uint16(0), // Initialize as 0
			},
			Body: []byte(fmt.Sprintf("ARQ test packet %d", i)),
		}

		if i == totalPackets {
			packet.Header.IsFinal = 1 // Set to 1 if it's the final packet
			log.Printf("This is the final packet (%d).", i)
		}

		log.Printf("Sending packet %d.", i)
		err := conv.sendPacket(conn, addr, uint32(i), packet.Body, DATA) // Assuming sendPacket handles serialization
		if err != nil {
			log.Printf("Error sending packet %d: %v", i, err)
		} else {
			log.Printf("Packet %d sent successfully.", i)
		}
	}

	log.Printf("ARQ mechanism test completed.")
}

// Updates the state of a packet based on whether an ACK has been received for it.
func (conv *conversation) updatePacketState(seqNum uint32, ackReceived bool) {
	conv.packetStatesMutex.Lock()         // Lock before accessing packetStates
	defer conv.packetStatesMutex.Unlock() // Ensure unlocking

	if state, exists := conv.packetStates[seqNum]; exists {
		state.AckReceived = ackReceived
	} else {
		conv.packetStates[seqNum] = &PacketState{AckReceived: ackReceived, LastSent: time.Now()}
	}
}
func uint16ToBool(value uint16) bool {
	return value != 0
}
