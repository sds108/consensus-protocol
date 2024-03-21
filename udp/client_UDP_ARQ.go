// Implements client that sends UDP packets to server.
// Simulates various scenarios for no packet loss, packet loss and has testing for
// the ARQ mechanism for reliable data transfer.

//*** Update and add more comments

// Version 1.5

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"strconv"
	"time"
)

var serverAddrStr = "127.0.0.1:8080"

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

	// Executes the ARQ tests for the following: no loss, with loss, and testing ARQ responses
	fmt.Println("1. Sending packets without loss:")
	sendPacketsWithoutLoss(conn, 5, conversation_id)

	fmt.Println("\n2. Sending packets with simulated loss:")
	sendPacketsWithLoss(conn, 5, 0.3, conversation_id) // 30% loss rate

	fmt.Println("\n3. Testing ARQ mechanism (observe server-side for behavior):")
	testARQMechanism(conn, 5, conversation_id) // Implemented based on the ARQ design
}

func sendPacketsWithoutLoss(conn *net.UDPConn, totalPackets int, conversation_id int) {
	for i := 1; i <= totalPackets; i++ {
		packet := Packet{ConversationId: strconv.Itoa(conversation_id), SequenceNumber: i, Data: fmt.Sprintf("No loss, packet %d", i), IsFinal: i == totalPackets}
		fmt.Printf("Sending packet: ConversationId=%s, SequenceNumber=%d, Data=%s, IsFinal=%t\n", packet.ConversationId, packet.SequenceNumber, packet.Data, packet.IsFinal)
		time.Sleep(5 * time.Second)
		if err := sendPacket(conn, packet); err != nil {
			log.Printf("Error sending packet %d: %v", i, err)
		}
	}
}

func sendPacketsWithLoss(conn *net.UDPConn, totalPackets int, lossRate float64, conversation_id int) {
	rand.Seed(time.Now().UnixNano())
	for i := 1; i <= totalPackets; i++ {
		if rand.Float64() >= lossRate {
			packet := Packet{ConversationId: strconv.Itoa(conversation_id), SequenceNumber: i, Data: fmt.Sprintf("With loss, packet %d", i), IsFinal: i == totalPackets}
			fmt.Printf("Sending packet: ConversationId=%s, SequenceNumber=%d, Data=%s, IsFinal=%t\n", packet.ConversationId, packet.SequenceNumber, packet.Data, packet.IsFinal)
			time.Sleep(5 * time.Second)
			if err := sendPacket(conn, packet); err != nil {
				log.Printf("Error sending packet %d: %v", i, err)
			}
		} else {
			fmt.Printf("Simulating loss for packet %d\n", i)
		}
	}
}

// **** Needs more testing ****

func testARQMechanism(conn *net.UDPConn, totalPackets int, conversation_id int) {
	fmt.Println("Testing ARQ mechanism by sending a sequence of packets with intentional loss...")

	sentPackets := make(map[int]Packet)
	// Simulates packet sending with potential loss
	for i := 1; i <= totalPackets; i++ {
		if rand.Float64() > 0.2 { // Adjust the loss rate as much as needed!!!
			packet := Packet{ConversationId: strconv.Itoa(conversation_id), SequenceNumber: i, Data: fmt.Sprintf("Test ARQ, packet %d", i), IsFinal: i == totalPackets}
			fmt.Printf("Sending packet: ConversationId=%s, SequenceNumber=%d, Data=%s, IsFinal=%t\n", packet.ConversationId, packet.SequenceNumber, packet.Data, packet.IsFinal)
			time.Sleep(5 * time.Second)
			if err := sendPacket(conn, packet); err != nil {
				log.Printf("Failed to send packet %d: %v", i, err)
			} else {
				sentPackets[i] = packet
			}
		} else {
			fmt.Printf("Simulating loss for packet %d\n", i)
		}
	}

	// Implements ACK receiving and packet re-sending here based on the ARQ design
	// its dependant on the implemented ARQ mechanism and how the acknowledgments
	// are handled and also how missing packets are detected!
}

func sendPacket(conn *net.UDPConn, packet Packet) error {
	packetData, err := json.Marshal(packet)
	if err != nil {
		return fmt.Errorf("failed to serialize packet: %v", err)
	}

	_, err = conn.Write(packetData)
	if err != nil {
		return fmt.Errorf("failed to send packet: %v", err)
	}

	fmt.Printf("Packet sent: {SequenceNumber=%d, Data=%s, IsFinal=%t}\n", packet.SequenceNumber, packet.Data, packet.IsFinal)
	return nil
}

func receiveACK(conn *net.UDPConn) (Packet, error) {
	buffer := make([]byte, 2048) // Adjust buffer size as much as needed!!
	readLen, _, err := conn.ReadFromUDP(buffer)
	if err != nil {
		return Packet{}, fmt.Errorf("error reading from UDP: %w", err)
	}

	var ackPacket Packet
	if err := json.Unmarshal(buffer[:readLen], &ackPacket); err != nil {
		return Packet{}, fmt.Errorf("error unmarshaling ACK packet: %w", err)
	}

	return ackPacket, nil
}

func resendPacket(conn *net.UDPConn, seqNum int, packets map[int]Packet) error {
	packet, exists := packets[seqNum]
	if !exists {
		return fmt.Errorf("packet with sequence number %d does not exist", seqNum)
	}
	packetData, err := json.Marshal(packet)
	if err != nil {
		return fmt.Errorf("failed to serialize packet for retransmission: %w", err)
	}

	if _, err := conn.Write(packetData); err != nil {
		return fmt.Errorf("failed to resend packet: %w", err)
	}

	fmt.Printf("Resent packet: SequenceNumber=%d, Data=%s\n", packet.SequenceNumber, packet.Data)
	return nil
}
