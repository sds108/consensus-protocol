// Implements client that sends UDP packets to server.
// Simulates various scenarios for no packet loss, packet loss, duplicates and has testing for
// the ARQ mechanism for reliable data transfer using selective repeat.

// Version 1.8
// Packet State structure added
// Sending packets with timers added
// Handling ACKs and NAKs added
// Memory management - once packet is sent, its state is stored in a map
// This allows for easy cleanup of the state when the packet is ACKed or NAKed

package main

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"sync"
	"time"
)

var serverAddrStr = "127.0.0.1:8080"

// Add global variables for window management
var (
	sendingWindowStart int = 1
	sendingWindowSize  int = 5 // Adjust based on the requirements!!
	packetStates           = make(map[int]*PacketState)
	sentPackets            = make(map[int]Pckt)
	ackReceived            = make(map[int]bool)
	ackReceivedLock    sync.Mutex
)

// Constants for packet types ACK and NAK
const (
	ACK  = 1 // Assuming 1 for ACK
	NACK = 2 // Automatically increments, assuming 2 for NAK
)

// Global variable for resend delay if packets not acknowledged
var resendDelay = 20 * time.Second

// PacketState holds the state of a packet, including whether an ACK was received and the last send time
type PacketState struct {
	AckReceived bool
	LastSent    time.Time  // Timestamp of the last time the packet was sent
	mutex       sync.Mutex // Mutex for concurrent access to the PacketState
}

// main initialises the UDP connection and runs tests for the ARQ mechanism.
func main() {
	serverAddr, err := net.ResolveUDPAddr("udp", serverAddrStr)
	if err != nil {
		log.Fatalf("Failed to resolve server address: %v", err)
	}

	conn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		log.Fatalf("Failed to dial UDP server: %v", err)
	}
	defer conn.Close()

	fmt.Println("My name is ", conn.RemoteAddr())
	fmt.Println("My name is ", conn.LocalAddr())

	conversation_id := rand.Int()

	go managePacketResends(conn)

	// Test without packet loss
	fmt.Println("1. Testing without packet loss...")
	sendPacketsWithoutLoss(conn, 5, conversation_id)

	// Test with simulated packet loss
	fmt.Println("\n2. Testing with simulated packet loss...")
	sendPacketsWithSimulatedLoss(conn, 5, conversation_id, 0.3) // 30% loss rate

	// Test handling of duplicate packets
	fmt.Println("\n3. Testing duplicate packet handling...")
	sendAndDuplicatePacket(conn, 1, conversation_id)

	fmt.Println("\n4. Testing ARQ mechanism (observe server-side for behavior):")
	testARQMechanism(conn, 5, conversation_id) // Implemented based on the ARQ design
}

// Tests the ARQ mechanism
func testARQMechanism(conn *net.UDPConn, totalPackets int, conversation_id int) {
	for {
		ackReceivedLock.Lock()

		// Move the sending window forward based on ACKs received.
		for ackReceived[sendingWindowStart] && sendingWindowStart <= totalPackets {
			sendingWindowStart++
		}
		ackReceivedLock.Unlock()

		// Check if all packets have been sent and acknowledged.
		if sendingWindowStart > totalPackets {
			fmt.Println("All packets acknowledged. Test complete.")
			os.Exit(0) // Exit the program after successful completion
		}

		// Send packets within the current window.
		for i := sendingWindowStart; i < sendingWindowStart+sendingWindowSize && i <= totalPackets; i++ {
			if _, sent := sentPackets[i]; !sent {
				sendPacket(conn, i, conversation_id, totalPackets)
			}
		}

		// Handle incoming ACKs/NACKs.
		handleIncomingACKsNACKs(conn)
	}
}

// sendPacket constructs and sends a packet to the server
func sendPacket(conn *net.UDPConn, seqNum int, conversation_id int, totalPackets int) {
	packet := Pckt{
		Header: PcktHeader{
			Magic:       magic,
			Checksum:    0, // Implement checksum calculation
			ConvID:      uint32(conversation_id),
			SequenceNum: uint32(seqNum),
			IsFinal:     boolToUint16(seqNum == totalPackets),
			Type:        Data,
		},
		Body: []byte(fmt.Sprintf("Test ARQ, packet %d", seqNum)),
	}
	sentPackets[seqNum] = packet

	packetState, exists := packetStates[int(packet.Header.SequenceNum)]
	if !exists {
		packetState = &PacketState{
			AckReceived: false,
			LastSent:    time.Now(),
			mutex:       sync.Mutex{},
		}
		packetStates[int(packet.Header.SequenceNum)] = packetState
	} else {
		packetState.LastSent = time.Now()
	}

	packetData, err := SerializePacket(packet) // Updated to use SerializePacket
	if err != nil {
		log.Printf("Error serializing packet: %v", err)
		return
	}

	_, err = conn.Write(packetData)
	if err != nil {
		log.Printf("Error sending packet: %v", err)
		return
	}

	log.Printf("Packet sent: {SequenceNumber=%d, Data=%s, IsFinal=%t}\n", packet.Header.SequenceNum, packet.Body, uint16ToBool(packet.Header.IsFinal))
}

// handleIncomingACKsNACKs listens for ACKs and NACKs from the server and processes them
func handleIncomingACKsNACKs(conn *net.UDPConn) {
	for {
		ackPacket, err := receiveACK(conn)
		if err != nil {
			if err == net.ErrClosed {
				break // Stop the loop if the connection is closed
			}
			log.Printf("Error receiving ACK/NACK: %v", err)
			continue
		}

		seqNum := int(ackPacket.Header.SequenceNum)
		ackReceivedLock.Lock()
		if ackReceived[seqNum] {
			log.Printf("Duplicate ACK/NACK received for packet %d. Ignoring.", seqNum)
			ackReceivedLock.Unlock()
			continue
		}
		if ackPacket.Header.Type == ACK {
			ackReceived[seqNum] = true
			log.Printf("ACK received for packet %d", seqNum)
			if packetState, exists := packetStates[seqNum]; exists {
				packetState.mutex.Lock()
				packetState.AckReceived = true
				packetState.mutex.Unlock()
				delete(packetStates, seqNum) // Cleanup packet state
			}
		} else if ackPacket.Header.Type == NACK {
			log.Printf("NACK received for packet %d, resending.", seqNum)
			resendPacket(conn, uint32(seqNum), sentPackets)
		}
		ackReceivedLock.Unlock()
	}

}

func sendPacketWithTimer(conn *net.UDPConn, packet Pckt, sentPackets map[int]Pckt) error {
	packetData, err := SerializePacket(packet) // Updated to use SerializePacket
	if err != nil {
		log.Printf("Error serializing packet: %v", err)
		return fmt.Errorf("failed to serialize packet: %v", err)
	}

	_, err = conn.Write(packetData)
	if err != nil {
		log.Printf("Error sending packet: %v", err)
		return fmt.Errorf("failed to send packet: %v", err)
	}

	log.Printf("Packet sent: {SequenceNumber=%d, Data=%s, IsFinal=%t}\n", packet.Header.SequenceNum, packet.Body, uint16ToBool(packet.Header.IsFinal))
	return nil
}

func startPacketTimer(seqNum int, packetState *PacketState, conn *net.UDPConn, packets map[int]Pckt) {
	time.Sleep(resendDelay) // Use the global variable for delay
	packetState.mutex.Lock()
	defer packetState.mutex.Unlock()
	if !packetState.AckReceived {
		fmt.Printf("Timer expired for packet %d, resending...\n", seqNum)
		resendPacket(conn, uint32(seqNum), packets)
	} else {
		fmt.Printf("ACK received for packet %d, no need to resend.\n", seqNum)
	}
}

func receiveACK(conn *net.UDPConn) (Pckt, error) {
	buffer := make([]byte, 2048) // Adjust buffer size as much as needed!!
	readLen, _, err := conn.ReadFromUDP(buffer)
	if err != nil {
		return Pckt{}, fmt.Errorf("error reading from UDP: %w", err)
	}

	ackPacket, err := DeserializePacket(buffer[:readLen]) // Updated to use DeserializePacket
	if err != nil {
		return Pckt{}, fmt.Errorf("error deserializing ACK packet: %w", err)
	}

	return *ackPacket, nil
}

func resendPacket(conn *net.UDPConn, seqNum uint32, packets map[int]Pckt) error {
	ackReceivedLock.Lock()
	defer ackReceivedLock.Unlock()
	if ackReceived[int(seqNum)] {
		// Packet already acknowledged, no need to resend
		return nil
	}

	packet, exists := packets[int(seqNum)]
	if !exists {
		return fmt.Errorf("packet with sequence number %d does not exist", seqNum)
	}
	packetData, err := SerializePacket(packet) // Updated to use SerializePacket
	if err != nil {
		return fmt.Errorf("failed to serialize packet for retransmission: %w", err)
	}

	if _, err := conn.Write(packetData); err != nil {
		return fmt.Errorf("failed to resend packet: %w", err)
	}

	log.Printf("Resent packet: SequenceNumber=%d, Data=%s\n", packet.Header.SequenceNum, packet.Body)
	return nil
}
func handleAck(conn *net.UDPConn, seqNum int, isNak bool) {
	ackReceivedLock.Lock()
	defer ackReceivedLock.Unlock()

	if packetState, exists := packetStates[seqNum]; exists {
		packetState.mutex.Lock()
		defer packetState.mutex.Unlock()
		if !isNak {
			ackReceived[seqNum] = true
			log.Printf("ACK received for packet %d, marking as acknowledged.", seqNum)
			packetState.AckReceived = true // Mark the packet as acknowledged
			delete(packetStates, seqNum)   // Optionally, remove the packet state to prevent resends
		} else {
			log.Printf("NAK received for packet %d, resending.", seqNum)
			resendPacket(conn, uint32(seqNum), sentPackets)
		}
	} else {
		log.Printf("Received ACK/NAK for unknown packet %d.", seqNum)
	}
}

func managePacketResends(conn *net.UDPConn) {
	ticker := time.NewTicker(resendDelay)
	defer ticker.Stop()

	for range ticker.C {
		ackReceivedLock.Lock()
		for seqNum, packetState := range packetStates {
			packetState.mutex.Lock()
			if !packetState.AckReceived && time.Since(packetState.LastSent) > resendDelay {
				fmt.Printf("Resend interval expired for packet %d, resending...\n", seqNum)
				resendPacket(conn, uint32(seqNum), sentPackets)
				packetState.LastSent = time.Now()
			}
			packetState.mutex.Unlock()
		}
		ackReceivedLock.Unlock()
	}
}

func cleanupPacketState(seqNum int) {
	delete(packetStates, seqNum) // Removes the reference, allowing garbage collection to free memory
}
func boolToUint16(value bool) uint16 {
	if value {
		return 1
	}
	return 0
}
func uint16ToBool(value uint16) bool {
	return value != 0
}

// Test functions for simulation
func sendAndDuplicatePacket(conn *net.UDPConn, seqNum int, conversation_id int) {
	// Construct a packet
	packet := Pckt{
		Header: PcktHeader{
			ConvID:      uint32(conversation_id),
			SequenceNum: uint32(seqNum),
			Type:        Data, // Assuming Data is a defined constant for packet type
		},
		Body: []byte(fmt.Sprintf("Test duplicate, packet %d", seqNum)),
	}

	// Serialize and send the packet
	packetData, _ := SerializePacket(packet)
	_, _ = conn.Write(packetData)
	log.Printf("Packet sent: SequenceNumber=%d", seqNum)

	// Immediately send the duplicate
	_, _ = conn.Write(packetData)
	log.Printf("Duplicate packet sent: SequenceNumber=%d", seqNum)
}

func sendPacketsWithoutLoss(conn *net.UDPConn, totalPackets int, conversation_id int) {
	for i := 1; i <= totalPackets; i++ {
		packet := Pckt{
			Header: PcktHeader{
				ConvID:      uint32(conversation_id),
				SequenceNum: uint32(i),
				Type:        Data, // Assuming Data is a defined constant for packet type
			},
			Body: []byte(fmt.Sprintf("Packet without loss %d", i)),
		}

		packetData, _ := SerializePacket(packet)
		_, _ = conn.Write(packetData)
		log.Printf("Packet sent without loss: SequenceNumber=%d", i)
	}
}

func sendPacketsWithSimulatedLoss(conn *net.UDPConn, totalPackets int, conversation_id int, lossRate float64) {
	rand.Seed(time.Now().UnixNano()) // Initialize a random number generator

	for i := 1; i <= totalPackets; i++ {
		if rand.Float64() > lossRate { // Simulate packet loss
			packet := Pckt{
				Header: PcktHeader{
					ConvID:      uint32(conversation_id),
					SequenceNum: uint32(i),
					Type:        Data,
				},
				Body: []byte(fmt.Sprintf("Packet with simulated loss %d", i)),
			}

			packetData, _ := SerializePacket(packet)
			_, _ = conn.Write(packetData)
			log.Printf("Packet sent with simulated loss: SequenceNumber=%d", i)
		} else {
			log.Printf("Simulating loss for packet: SequenceNumber=%d", i)
		}
	}
}

func testDuplicateHandling(conn *net.UDPConn, conversation_id int) {
	// Send a unique packet
	sendAndDuplicatePacket(conn, 1, conversation_id)

	// Send another unique packet followed by a duplicate
	sendAndDuplicatePacket(conn, 2, conversation_id)

	// Can continue with more tests if needed?
}
