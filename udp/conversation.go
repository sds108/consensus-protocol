package main // Declares that this file is part of the main package.

import (
	// Import the fmt package for printing.
	"errors"
	"fmt"
	"log"
	"net"
	"sync" // Import the sync package for mutexes.
	"time"

	"github.com/google/uuid"
)

// Sliding Window Structure (holds packets waiting to be sent, and all other SR attributes)
type sliding_window struct {
	// Outgoing map (holds pointers to packet objects to be sent)
	// (first key is Packet Number, second key is Sequence Number)
	outgoing      map[uint32]*Pckt
	outgoing_lock sync.Mutex

	// Sliding Window (SR) attributes
	windowStart uint32
	windowSize  uint32
	nextPcktNum uint32
}

// Handles the SR functionality for incoming packets
type receiving_window struct {
	// Buffer for incoming packets
	// the key is the packet number
	incoming         map[uint32]*Pckt
	incoming_lock    sync.Mutex
	lastPcktReceived uint32
}

// Structure container, connection free and instead dependent on the conversation's ID
type conversation struct {
	// Conversation ID
	conversation_id uint32

	// SR Receiver Structure
	receiver *receiving_window

	// SR Sender Structure
	sender *sliding_window

	// UDP Address of the Node Corresponding to this Conversation
	conversation_addr *net.UDPAddr

	conversation_features []uint16
}

// newConversation creates a new conversation instance
func newConversation(conversation_id uint32, conv_addr *net.UDPAddr) *conversation {
	return &conversation{
		conversation_id:   conversation_id,
		conversation_addr: conv_addr,
		receiver: &receiving_window{
			incoming:         make(map[uint32]*Pckt),
			lastPcktReceived: 0,
		},
		sender: &sliding_window{
			outgoing:    make(map[uint32]*Pckt),
			windowStart: 0,
			windowSize:  5,
			nextPcktNum: 0,
		},
	}
}

func (conv *conversation) updateConversationAddr(conv_addr *net.UDPAddr) {
	if conv_addr != conv.conversation_addr {
		conv.conversation_addr = conv_addr
	}
}

// startConversation starts all of the Routines associated with said conversation
func (conv *conversation) looper() {
	for true {
		conv.incomingProcessor()
		conv.sendWindowPackets()
		conv.checkForRetransmissions()
		time.Sleep(3 * time.Millisecond)
	}
}

// ARQ_Receive handles incoming packets, checks for duplicates, and sends ACKs/NAKs, updates receivedPackets
func (conv *conversation) ARQ_Receive(conn *net.UDPConn, addr *net.UDPAddr, pckt Pckt) {
	//fmt.Printf("Received packet: ConversationId=%d, PacketNumber=%d, SequenceNumber=%d, Data=%s, IsFinal=%t\n", pckt.Header.ConvID, pckt.Header.SequenceNum, pckt.Header.SequenceNum, string(pckt.Body), pckt.Header.IsFinal != 0)

	switch pckt.Header.Type {
	case DATA:
		{
			// Send ACK for packet
			conv.sendACK(pckt.Header.PacketNum, pckt.Header.SequenceNum)

			// Lock Receiver
			conv.receiver.incoming_lock.Lock()
			defer conv.receiver.incoming_lock.Unlock()

			// Check if single fragment packet
			if pckt.Header.IsFinal == 0 || pckt.Header.SequenceNum > 0 {
				// drop fragment packet
				conv.receiver.lastPcktReceived = pckt.Header.PacketNum
				fmt.Printf("Rejecting Multi Fragment Packet %d.\n", pckt.Header.PacketNum)
				return
			}

			// Check if duplicate
			if _, exists := conv.receiver.incoming[pckt.Header.PacketNum]; !exists {
				//fmt.Println(pckt)
				conv.receiver.incoming[pckt.Header.PacketNum] = &pckt
				conv.receiver.lastPcktReceived = pckt.Header.PacketNum
			} else {
				fmt.Printf("Duplicate packet received: %d.\n", pckt.Header.PacketNum)
			}

			// Updates the highest sequence number if required
			if pckt.Header.PacketNum > conv.receiver.lastPcktReceived {
				// check for gap
				if pckt.Header.PacketNum > conv.receiver.lastPcktReceived+1 {
					for i := conv.receiver.lastPcktReceived + 1; i < pckt.Header.PacketNum; i++ {
						fmt.Printf("Packet %d: %d does not exist, sending NACK\n", pckt.Header.PacketNum, pckt.Header.SequenceNum)
						conv.sendNAK(i, 0)
					}
				}
				conv.receiver.lastPcktReceived = pckt.Header.PacketNum
			}
		}

	case ACK:
		{
			fmt.Printf("Got an ACK for Packet %d.\n", pckt.Header.PacketNum)

			// Lock Sender
			conv.sender.outgoing_lock.Lock()
			defer conv.sender.outgoing_lock.Unlock()

			// Make sure outgoing packet exists
			if _, exists := conv.sender.outgoing[pckt.Header.PacketNum]; !exists {
				log.Printf("Packet %d does not exist, cannot Ack.", pckt.Header.PacketNum)
				return // Drop Ack
			}

			// Set Ack received state to true
			conv.sender.outgoing[pckt.Header.PacketNum].AckReceived = true
		}

	case NAK:
		{
			fmt.Printf("Got a NACK for Packet %d.\n", pckt.Header.PacketNum)

			// Lock Sender
			conv.sender.outgoing_lock.Lock()
			defer conv.sender.outgoing_lock.Unlock()

			// Make sure outgoing packet exists
			if _, exists := conv.sender.outgoing[pckt.Header.PacketNum]; !exists {
				log.Printf("Packet %d does not exist, cannot resend for Nack.", pckt.Header.PacketNum)
				return // Drop Nack
			}

			// Make sure packet wasn't Acked before the lock
			if conv.sender.outgoing[pckt.Header.PacketNum].AckReceived == true {
				log.Printf("Packet %d already Acked, won't resend.", pckt.Header.PacketNum)
				return // Drop Nack
			}

			// Resend Packet
			conv.sendPacket(conv.sender.outgoing[pckt.Header.PacketNum])
		}

	case SYN:
		{
			fmt.Printf("Got a SYN\n")
			// Instantly respond with SYN_ACK
			conv.sendSYN_ACK()
		}

	case SYN_ACK:
		{
			fmt.Printf("Got a SYN ACK\n")
			// Do nothing for now
			conv.sendHello()
		}

	default:
		{
			fmt.Printf("Received an Unknown Packet Type\n")
			return // Drop packet
		}
	}
}

// sendHello sends a Hello Packet
func (conv *conversation) sendHello() {
	// Create the Hello Struct for the body of the Packet
	helloBody := PcktHello{
		DataID:      hello_c2s,
		Version:     0,
		NumFeatures: uint16(len(my_features)),
		Features:    my_features,
	}

	helloBody_bytes, err := SerializeHello(&helloBody)
	if err != nil {
		return
	}

	// Lock outgoing
	conv.sender.outgoing_lock.Lock()
	defer conv.sender.outgoing_lock.Unlock()

	helloPacket := Pckt{
		Header: PcktHeader{
			Magic:       MAGIC_CONST,
			Checksum:    0,
			ConvID:      conversation_id_self,
			PacketNum:   conv.sender.nextPcktNum,
			SequenceNum: 0,
			Type:        DATA,
			IsFinal:     1,
		},
		Body: make([]byte, 0),
	}

	helloPacket.Body = append(helloPacket.Body, helloBody_bytes...)

	// Make Sure we're not exceeding Maximum Packet Size
	if len(helloPacket.Body) > MAX_PCKT_SIZE {
		// Drop this packet
		return
	} else {
		// Append to outgoing
		conv.sender.outgoing[helloPacket.Header.PacketNum] = &helloPacket

		// Increment next Packet Number
		conv.sender.nextPcktNum += 1
	}
}

// sendHelloBack sends a Hello Back Packet
func (conv *conversation) sendHelloResonse() {
	// Create the Hello Struct for the body of the Packet
	helloBackBody := PcktHello{
		DataID:      hello_back_s2c,
		Version:     0,
		NumFeatures: uint16(len(my_features)),
		Features:    my_features,
	}

	helloBackBody_bytes, err := SerializeHello(&helloBackBody)
	if err != nil {
		return
	}

	// Lock outgoing
	conv.sender.outgoing_lock.Lock()
	defer conv.sender.outgoing_lock.Unlock()

	helloBackPacket := Pckt{
		Header: PcktHeader{
			Magic:       MAGIC_CONST,
			Checksum:    0,
			ConvID:      conversation_id_self,
			PacketNum:   conv.sender.nextPcktNum,
			SequenceNum: 0,
			Type:        DATA,
			IsFinal:     1,
		},
		Body: make([]byte, 0),
	}

	helloBackPacket.Body = append(helloBackPacket.Body, helloBackBody_bytes...)

	// Make Sure we're not exceeding Maximum Packet Size
	if len(helloBackPacket.Body) > MAX_PCKT_SIZE {
		// Drop this packet
		return
	} else {
		// Append to outgoing
		conv.sender.outgoing[helloBackPacket.Header.PacketNum] = &helloBackPacket

		// Increment next Packet Number
		conv.sender.nextPcktNum += 1
	}
}

func (conv *conversation) sendVoteRequestToServer(question string) {
	// Create the Vote Request Struct for the body of the Packet
	voteid, err := uuid.NewUUID()
	if err != nil {
		log.Print(err)
		return
	}

	voteReqBody := PcktVoteRequest{
		DataID:         vote_c2s_request_vote,
		VoteID:         voteid,
		QuestionLength: uint32(len(question)),
		Question:       question,
	}

	voteReqBody_bytes, err := SerializeVoteRequest(&voteReqBody)
	if err != nil {
		log.Println(err)
		return
	}

	// Lock outgoing
	conv.sender.outgoing_lock.Lock()
	defer conv.sender.outgoing_lock.Unlock()

	voteReqPacket := Pckt{
		Header: PcktHeader{
			Magic:       MAGIC_CONST,
			Checksum:    0,
			ConvID:      conversation_id_self,
			PacketNum:   conv.sender.nextPcktNum,
			SequenceNum: 0,
			Type:        DATA,
			IsFinal:     1,
		},
		Body: make([]byte, 0),
	}

	voteReqPacket.Body = append(voteReqPacket.Body, voteReqBody_bytes...)

	// Make Sure we're not exceeding Maximum Packet Size
	if len(voteReqPacket.Body) > MAX_PCKT_SIZE {
		// Drop this packet
		return
	} else {
		// Append to outgoing
		conv.sender.outgoing[voteReqPacket.Header.PacketNum] = &voteReqPacket

		// Increment next Packet Number
		conv.sender.nextPcktNum += 1
	}
}

func (conv *conversation) sendVoteBroadcastToClient(h_ref *host_referendum) {
	// Create the Vote Broadcast Struct for the body of the Packet
	voteBrBody := PcktVoteRequest{
		DataID:         vote_s2c_broadcast_question,
		VoteID:         h_ref.VoteID,
		QuestionLength: uint32(len(h_ref.Question)),
		Question:       h_ref.Question,
	}

	voteBrBody_bytes, err := SerializeVoteRequest(&voteBrBody)
	if err != nil {
		return
	}

	// Lock outgoing
	conv.sender.outgoing_lock.Lock()
	defer conv.sender.outgoing_lock.Unlock()

	voteBrPacket := Pckt{
		Header: PcktHeader{
			Magic:       MAGIC_CONST,
			Checksum:    0,
			ConvID:      conversation_id_self,
			PacketNum:   conv.sender.nextPcktNum,
			SequenceNum: 0,
			Type:        DATA,
			IsFinal:     1,
		},
		Body: make([]byte, 0),
	}

	voteBrPacket.Body = append(voteBrPacket.Body, voteBrBody_bytes...)

	// Make Sure we're not exceeding Maximum Packet Size
	if len(voteBrPacket.Body) > MAX_PCKT_SIZE {
		// Drop this packet
		return
	} else {
		// Append to outgoing
		conv.sender.outgoing[voteBrPacket.Header.PacketNum] = &voteBrPacket

		// Increment next Packet Number
		conv.sender.nextPcktNum += 1
	}
}

func (conv *conversation) sendResponseToServer(c_ref *client_referendum) {
	// Create the Vote Broadcast Struct for the body of the Packet
	voteResBody := PcktVoteResponse{
		DataID:   vote_c2s_response_to_question,
		VoteID:   c_ref.VoteID,
		Response: c_ref.result,
	}

	voteResBody_bytes, err := SerializeVoteResponse(&voteResBody)
	if err != nil {
		return
	}

	// Lock outgoing
	conv.sender.outgoing_lock.Lock()
	defer conv.sender.outgoing_lock.Unlock()

	voteResPacket := Pckt{
		Header: PcktHeader{
			Magic:       MAGIC_CONST,
			Checksum:    0,
			ConvID:      conversation_id_self,
			PacketNum:   conv.sender.nextPcktNum,
			SequenceNum: 0,
			Type:        DATA,
			IsFinal:     1,
		},
		Body: make([]byte, 0),
	}

	voteResPacket.Body = append(voteResPacket.Body, voteResBody_bytes...)

	// Make Sure we're not exceeding Maximum Packet Size
	if len(voteResPacket.Body) > MAX_PCKT_SIZE {
		// Drop this packet
		return
	} else {
		// Append to outgoing
		conv.sender.outgoing[voteResPacket.Header.PacketNum] = &voteResPacket

		// Increment next Packet Number
		conv.sender.nextPcktNum += 1
	}
}

func (conv *conversation) sendResultBroadcastToClient(h_ref *host_referendum) {
	// Create the Vote Broadcast Struct for the body of the Packet
	voteResBrBody := PcktVoteResponse{
		DataID:   vote_s2c_broadcast_result,
		VoteID:   h_ref.VoteID,
		Response: h_ref.result,
	}

	voteResBrBody_bytes, err := SerializeVoteResponse(&voteResBrBody)
	if err != nil {
		return
	}

	// Lock outgoing
	conv.sender.outgoing_lock.Lock()
	defer conv.sender.outgoing_lock.Unlock()

	voteResBrPacket := Pckt{
		Header: PcktHeader{
			Magic:       MAGIC_CONST,
			Checksum:    0,
			ConvID:      conversation_id_self,
			PacketNum:   conv.sender.nextPcktNum,
			SequenceNum: 0,
			Type:        DATA,
			IsFinal:     1,
		},
		Body: make([]byte, 0),
	}

	voteResBrPacket.Body = append(voteResBrPacket.Body, voteResBrBody_bytes...)

	// Make Sure we're not exceeding Maximum Packet Size
	if len(voteResBrPacket.Body) > MAX_PCKT_SIZE {
		// Drop this packet
		return
	} else {
		// Append to outgoing
		conv.sender.outgoing[voteResBrPacket.Header.PacketNum] = &voteResBrPacket

		// Increment next Packet Number
		conv.sender.nextPcktNum += 1
	}
}

// sendSYN sends a SYN
func (conv *conversation) sendSYN() {
	synPacket := Pckt{
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

	conv.sendPacket(&synPacket)
}

// sendSYN sends a SYN
func (conv *conversation) sendSYN_ACK() {
	syn_ackPacket := Pckt{
		Header: PcktHeader{
			Magic:       MAGIC_CONST,
			Checksum:    0,
			ConvID:      conversation_id_self,
			PacketNum:   0,
			SequenceNum: 0,
			Type:        SYN_ACK,
			IsFinal:     1,
		},
		Body: []byte{},
	}

	conv.sendPacket(&syn_ackPacket)
}

// sendNAK sends a NAK for a missing packet
func (conv *conversation) sendNAK(missingPcktNum uint32, missingSeqNum uint32) {
	nakPacket := Pckt{
		Header: PcktHeader{
			Magic:       MAGIC_CONST,
			Checksum:    0,
			ConvID:      conversation_id_self,
			PacketNum:   missingPcktNum,
			SequenceNum: missingSeqNum,
			Type:        NAK,
			IsFinal:     1,
		},
		Body: []byte{},
	}

	conv.sendPacket(&nakPacket)
}

// sendACK sends an ACK for a received packet
func (conv *conversation) sendACK(pcktNum uint32, seqNum uint32) {
	ackPacket := Pckt{
		Header: PcktHeader{
			Magic:       MAGIC_CONST,
			Checksum:    0,
			ConvID:      conversation_id_self,
			PacketNum:   pcktNum,
			SequenceNum: seqNum,
			Type:        ACK,
			IsFinal:     1,
		},
		Body: []byte{},
	}

	conv.sendPacket(&ackPacket)
}

// Sends a packet and updates its state in the packetStates map, marks it as sent and records the sending time.
func (conv *conversation) sendPacket(pckt *Pckt) error {
	// Send Packet
	if err := sendUDP(conv.conversation_addr, pckt); err != nil {
		return errors.New("Packet Couldn't Send")
	}

	if pckt.Header.Type == DATA {
		// Set Ack received state to false
		pckt.AckReceived = false

		// Reset Last Sent Timestamp
		pckt.LastSent = time.Now()
	}

	//log.Printf("Packet sent: ConversationId=%d, SequenceNumber=%d, Type=%d, IsFinal=%t\n", conv.conversation_id, seqNum, packetType, uint16ToBool(packet.Header.IsFinal))

	return nil
}

// Manages the sending window by sending packets within the window size, ensures packets are sent in sequence
func (conv *conversation) sendWindowPackets() {
	conv.sender.outgoing_lock.Lock()

	conv.moveWindow()

	for i := conv.sender.windowStart; i < conv.sender.windowStart+conv.sender.windowSize; i++ {
		// Make sure packet exists in outgoing
		if _, exists := conv.sender.outgoing[i]; exists {
			// Make sure it's not a NULL pointer
			if conv.sender.outgoing[i] != nil {
				// Make sure we haven't received an ACK for it before this function
				if !conv.sender.outgoing[i].AckReceived {
					conv.sendPacket(conv.sender.outgoing[i])
				}
			} else {
				log.Printf("NULL pointer in window slice")
			}
		} else {
			//log.Printf("Send Window Packets: Reached the end of the window")
			break
		}
	}

	conv.sender.outgoing_lock.Unlock()
}

// Handles the window sliding as packets are acknowledged.
func (conv *conversation) moveWindow() {
	var original_windowStart uint32 = conv.sender.windowStart

	for i := original_windowStart; i < original_windowStart+conv.sender.windowSize; i++ {
		// Make sure packet exists in outgoing
		if _, exists := conv.sender.outgoing[i]; exists {
			// Make sure it's not a NULL pointer
			if conv.sender.outgoing[i] != nil {
				if conv.sender.outgoing[i].AckReceived {
					// Move window if this packet is ACKed
					conv.sender.windowStart += 1

					// Delete Acked Packet
					delete(conv.sender.outgoing, i)
				} else {
					// Stop moving window
					break
				}
			} else {
				log.Printf("NULL pointer in outgoing.\n")
			}
		} else {
			//log.Printf("Packet %d doesn't exist in outgoing.\n", i)
		}
	}
}

// Implementation of timers for managing packet resends, periodically checks the sliding window and
// resends any packets that have not been acknowledged after a certain time interval.
func (conv *conversation) checkForRetransmissions() {
	conv.sender.outgoing_lock.Lock() // Ensure thread-safe access to sender

	if len(conv.sender.outgoing) > 0 {
		for i := conv.sender.windowStart; i < conv.sender.windowStart+conv.sender.windowSize; i++ {
			// Make sure packet exists in outgoing
			if _, exists := conv.sender.outgoing[i]; exists {
				if conv.sender.outgoing[i] != nil {
					if !conv.sender.outgoing[i].AckReceived && time.Since(conv.sender.outgoing[i].LastSent) > 1000*time.Millisecond {
						fmt.Println("Resending %d.\n", conv.sender.outgoing[i].Header.PacketNum)
						conv.sendPacket(conv.sender.outgoing[i])
					}
				} else {
					log.Printf("NULL pointer in outgoing.\n")
				}
			} else {
				//log.Printf("Packet %d doesn't exist in outgoing.\n", i)
			}
		}
	}

	conv.sender.outgoing_lock.Unlock()
}

func (conv *conversation) incomingProcessor() {
	// Lock incoming
	conv.receiver.incoming_lock.Lock()
	defer conv.receiver.incoming_lock.Unlock()

	// Check if there are any packets waiting to be processed
	if len(conv.receiver.incoming) > 0 {

		var minPcktNum uint32

		for pcktNum := range conv.receiver.incoming {
			minPcktNum = pcktNum
			break
		}

		// Make sure it actually exists
		if _, exists := conv.receiver.incoming[minPcktNum]; !exists {
			log.Printf("Packet %d does not exist in incoming.", minPcktNum)
			return
		} else {
			// Delete Packet from incoming after we're done with it
			defer delete(conv.receiver.incoming, minPcktNum)
		}

		// Process the next packet
		DataID, err := DeserializeDataID(conv.receiver.incoming[minPcktNum].Body[:2])
		if err != nil {
			log.Printf("Couldn't get Data ID")
			return
		}

		switch DataID {
		case hello_c2s:
			{
				log.Printf("Got a hello\n")
				hello, err := DeserializeHello(conv.receiver.incoming[minPcktNum].Body)
				if err != nil {
					log.Printf("Could't Deserialize Hello Packet")
					return
				}

				conv.conversation_features = hello.Features

				// Send a Hello Back
				conv.sendHelloResonse()

			}

		case hello_back_s2c:
			{
				log.Printf("Got a hello back\n")
				hello_response, err := DeserializeHello(conv.receiver.incoming[minPcktNum].Body)
				if err != nil {
					log.Printf("Could't Deserialize Hello Response Packet")
					return
				}

				conv.conversation_features = hello_response.Features

				// Try send request
				time.Sleep(10 * time.Second)
				conv.sendVoteRequestToServer("2+2==5=1")
			}

		case vote_c2s_request_vote:
			{
				vote_request, err := DeserializeVoteRequest(conv.receiver.incoming[minPcktNum].Body)
				if err != nil {
					log.Printf("Could't Deserialize Vote Request from Client Packet")
					return
				}

				fmt.Print(vote_request.Question)

				// As Server, Begin a vote
				ref_manager.create_referendum_from_client_request(vote_request)
			}

		case vote_s2c_broadcast_question:
			{
				vote_broadcast_question, err := DeserializeVoteRequest(conv.receiver.incoming[minPcktNum].Body)
				if err != nil {
					log.Printf("Could't Deserialize Vote Broadcast Question from Server Packet")
					return
				}

				fmt.Print(vote_broadcast_question.Question)

				// As a Client, Process Question and send your response back to server
				ref_manager.handle_new_question_from_server(vote_broadcast_question, conv)
			}

		case vote_c2s_response_to_question:
			{
				vote_response, err := DeserializeVoteResponse(conv.receiver.incoming[minPcktNum].Body)
				if err != nil {
					log.Printf("Could't Deserialize Vote Response from Client Packet")
					return
				}

				fmt.Print(vote_response.Response)

				// As Server, log Client response
				ref_manager.handle_response_from_client(vote_response, conv)
			}

		case vote_s2c_broadcast_result:
			{
				vote_broadcast_result, err := DeserializeVoteResponse(conv.receiver.incoming[minPcktNum].Body)
				if err != nil {
					log.Printf("Could't Deserialize Vote Broadcast Result from Server Packet")
					return
				}

				fmt.Print(vote_broadcast_result.Response)

				// As Client, overwrite your own response to the Question if you got it wrong
				ref_manager.handle_result_from_server(vote_broadcast_result)
			}

		default:
			{
				log.Printf("Received an Unknown Data ID")
			}
		}
	}
}

/*
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
*/

func uint16ToBool(value uint16) bool {
	return value != 0
}
