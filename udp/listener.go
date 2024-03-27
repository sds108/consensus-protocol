package main

import (
	//"fmt"
	"log"
	"net"
)

func listener(udpPort string) {
	// Resolve UDP Address to listen at
	addr, err := net.ResolveUDPAddr("udp", ":"+udpPort)
	if err != nil {
		log.Fatal(err)
	}

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
	packet, err := DeserializePacket(raw_packet)
	if err != nil {
		log.Printf("handleIncomingPackets: DeserializeHeader returned Error\n")
		return
	}

	conversations_lock.Lock()
	conversationRef, exists := conversations[packet.Header.ConvID]
	if !exists {
		conversationRef = newConversation(packet.Header.ConvID)
		conversations[packet.Header.ConvID] = conversationRef
	}
	conversations_lock.Unlock()

	conversationRef.ARQ_Receive(conn, addr, *packet)
}
