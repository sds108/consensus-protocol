// Focuses on the server side logic.
// Includes listening for incoming packets and handlin them.

// Version 1.3

package main

import (
	"fmt"
	"log"
	"net"
)

func listener(udpPort string) {
	// Resolve UDP Address to listen at
	addr, err := net.ResolveUDPAddr("udp", ":"+udpPort)
	if err != nil {
		log.Fatal(err)
	}

	// Start Connection Socket on the UDP Address
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatal(err)
	}

	defer conn.Close()
	log.Printf("UDP server listening on port %s", udpPort)

	// Listen Continuously
	for {
		buffer := make([]byte, 4096)
		n, addr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Println("Error reading from UDP:", err)
			continue
		}

		go handleIncomingPackets(conn, addr, buffer[:n])
	}
}

func handleIncomingPackets(conn *net.UDPConn, addr *net.UDPAddr, raw_packet []byte) {
	// Make sure Data is at least 20 Bytes
	if len(raw_packet) < 20 {
		log.Printf("handleIncomingPackets: insufficient packet size\n")
		return
	}

	// Magic and Checksum check, if this fails, you would drop the packet
	verify_packet, err := VerifyPacket(raw_packet)
	if err != nil {
		log.Printf("handleIncomingPackets: VerifyPacket returned Error\n")
		return
	}

	if !verify_packet {
		log.Printf("handleIncomingPackets: VerifyPacket returned False\n")
		return
	}

	// Deserialize the Header
	pcktheader, err := DeserializeHeader(raw_packet)
	if err != nil {
		log.Printf("handleIncomingPackets: DeserializeHeader returned Error\n")
		return
	}

	//fmt.Printf("Received packet: ConversationId=%s, SequenceNumber=%d, Data=%s, IsFinal=%t\n", packet.ConversationId, packet.SequenceNumber, packet.Data, packet.IsFinal)

	// Create Packet
	pckt := Pckt{*pcktheader, raw_packet[20:]}

	// Find Corresponding Conversation
	conversation_reference, exists := (conversations_map)[pcktheader.ConvID]

	// If it exists, send the packet to the corresponding conversation
	if exists {
		conversation_reference.ARQ_Receive(conn, addr, pckt)
	} else {
		// Else create a new conversation
		conversations_map_lock.Lock()
		conversations_map[pcktheader.ConvID] = newConversation(pcktheader.ConvID)
		conversations_map_lock.Unlock()

		// and now send it there
		conversations_map[pcktheader.ConvID].ARQ_Receive(conn, addr, pckt)

	}

	// Print size of conversations list
	fmt.Printf("Conversations: %d\n", len(conversations_map))
}
