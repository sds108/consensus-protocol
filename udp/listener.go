package main

import (
	//"fmt"
	"log"
	"net"
	"sync"
)

var (
	udpPort       = "8080"
	conversations = make(map[uint32]*conversation)
	convMutex     sync.Mutex // Protects access to the conversations map
)

func listener() {
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

func handleIncomingPackets(conn *net.UDPConn, addr *net.UDPAddr, data []byte) {
	if len(data) < 20 {
		log.Printf("Packet size below 20 Bytes???\n")
		return
	}

	packet, err := DeserializePacket(data)
	if err != nil {
		log.Printf("Error deserializing packet: %v", err)
		return
	}

	convMutex.Lock()
	conversationRef, exists := conversations[packet.Header.ConvID]
	if !exists {
		conversationRef = newConversation(packet.Header.ConvID)
		conversations[packet.Header.ConvID] = conversationRef
	}
	convMutex.Unlock()

	conversationRef.ARQ_Receive(conn, addr, *packet)
}

func main() {
	listener()
}
