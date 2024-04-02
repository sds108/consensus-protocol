package main // Declares that this file is part of the main package.

import (
	// Import the fmt package for printing.
	"errors"
	"fmt"
	"log"
	"net"
	"sync" // Import the sync package for mutexes.
	"time"
)

// Sliding Window Structure (holds packets waiting to be sent, and all other SR attributes)
type sliding_window struct {
	// Outgoing map (holds pointers to packet objects to be sent)
	// (first key is Packet Number, second key is Sequence Number)
	outgoing      map[uint32](map[uint32]*Pckt)
	outgoing_lock sync.Mutex

	// Sliding Window (SR) attributes
	window      []*Pckt
	windowStart uint32
	windowSize  uint32
	nextPcktNum uint32

	// Transmission Ticker
	transmission_ticker *time.Ticker

	// Retransmission Ticker
	retransmission_ticker *time.Ticker
}

// Handles the SR functionality for incoming packets
type receiving_window struct {
	// Buffer for incoming packets
	// (first key is Packet Number, second key is Sequence Number)
	incoming      map[uint32](map[uint32]*Pckt)
	incoming_lock sync.Mutex
	// (key is Packet Number, Values are last sequence received for each packet)
	lastSeqReceived map[uint32]uint32
}

// Holds reassembled packets ready for the incomming processor to take care of
type packet_basket struct {
	// The basket holds the body of each complete packet
	basket      map[uint32]([]byte)
	done        map[uint32]bool
	basket_lock sync.Mutex
	lastPcktNum uint32

	// Packet Processing Ticker
	processing_ticker *time.Ticker
}

// Structure container, connection free and instead dependent on the conversation's ID
type conversation struct {
	// Conversation ID
	conversation_id uint32

	// SR Receiver Structure
	receiver *receiving_window

	// Basket for reassembled packets
	bskt *packet_basket

	// SR Sender Structure
	sender *sliding_window

	// UDP Address of the Node Corresponding to this Conversation
	conversation_addr *net.UDPAddr
}

// newConversation creates a new conversation instance
func newConversation(conversation_id uint32, conv_addr *net.UDPAddr) *conversation {
	return &conversation{
		conversation_id:   conversation_id,
		conversation_addr: conv_addr,
		receiver: &receiving_window{
			incoming:        make(map[uint32](map[uint32]*Pckt)),
			lastSeqReceived: make(map[uint32]uint32),
		},
		bskt: &packet_basket{
			basket:            make(map[uint32][]byte),
			done:              make(map[uint32]bool),
			lastPcktNum:       0,
			processing_ticker: time.NewTicker(200 * time.Millisecond),
		},
		sender: &sliding_window{
			outgoing:              make(map[uint32](map[uint32]*Pckt)),
			window:                make([]*Pckt, 0),
			windowStart:           0,
			windowSize:            5,
			nextPcktNum:           0,
			transmission_ticker:   time.NewTicker(500 * time.Millisecond),
			retransmission_ticker: time.NewTicker(1000 * time.Millisecond),
		},
	}
}

// startConversation starts all of the Go Routines associated with said conversation
func (conv *conversation) startConversation() {
	// Start Incoming Processor
	// go conv.incomingProcessor()

	// // Start Outgoing Processor
	// go conv.sendWindowPackets()

	// // Start Retransmission Checker
	// go conv.checkForRetransmissions()

	go conv.looper()
}

func (conv *conversation) looper() {
	for true {
		//fmt.Println("looping")
		conv.incomingProcessor()
		conv.sendWindowPackets()
		conv.checkForRetransmissions()
		time.Sleep(10 * time.Millisecond)
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

			fmt.Println(pckt)

			// Lock Receiver
			conv.receiver.incoming_lock.Lock()

			// Check if sub map exists
			if _, exists := conv.receiver.incoming[pckt.Header.PacketNum]; !exists {
				conv.receiver.incoming[pckt.Header.PacketNum] = make(map[uint32]*Pckt)
				conv.receiver.lastSeqReceived[pckt.Header.PacketNum] = 0
			}

			// Check for duplicate packets
			if _, exists := conv.receiver.incoming[pckt.Header.PacketNum][pckt.Header.SequenceNum]; exists {
				fmt.Printf("Duplicate packet received: %d.\n", pckt.Header.SequenceNum)
				//conv.receiver.incoming_lock.Unlock()
				//return // Return early to avoid any window movement or state change
			} else {
				// Store Packet Pointer to Incoming Buffer inside the receiver
				conv.receiver.incoming[pckt.Header.PacketNum][pckt.Header.SequenceNum] = &pckt
			}

			// Updates the highest sequence number if required
			if pckt.Header.SequenceNum > conv.receiver.lastSeqReceived[pckt.Header.PacketNum] {
				if pckt.Header.SequenceNum > conv.receiver.lastSeqReceived[pckt.Header.PacketNum]+1 {
					for i := conv.receiver.lastSeqReceived[pckt.Header.PacketNum] + 1; i < pckt.Header.SequenceNum; i++ {
						fmt.Printf("Packet %d: %d does not exist, sending NACK\n", pckt.Header.PacketNum, pckt.Header.SequenceNum)
						conv.sendNAK(pckt.Header.PacketNum, i)
					}
				}
				conv.receiver.lastSeqReceived[pckt.Header.PacketNum] = pckt.Header.SequenceNum
			}

			// Check if ready to be reassembled, send Nacks for gaps
			if conv.receiver.incoming[pckt.Header.PacketNum][conv.receiver.lastSeqReceived[pckt.Header.PacketNum]].Header.IsFinal == uint16(1) {
				// Boolean will remain true if all segments found
				fullPacket := true

				// Check for gaps
				for i := uint32(0); i <= uint32(conv.receiver.lastSeqReceived[pckt.Header.PacketNum]); i++ {
					// Make sure Packet Segment exists
					if _, exists := conv.receiver.incoming[pckt.Header.PacketNum][i]; !exists {
						fullPacket = false
						//conv.sendNAK(pckt.Header.PacketNum, i)
						break
					}
				}

				// Reassemble Packet
				if fullPacket {
					// Lock Basket
					//conv.bskt.basket_lock.Lock()

					// Check if Full Packet already exists in the basket
					if _, exists := conv.bskt.basket[pckt.Header.PacketNum]; exists {
						fmt.Printf("Packet %d already present in basket\n", pckt.Header.PacketNum)
					} else {
						conv.bskt.basket[pckt.Header.PacketNum] = make([]byte, 0)
					}

					if conv.bskt.done[pckt.Header.PacketNum] == false {
						conv.bskt.done[pckt.Header.PacketNum] = true
						for seqNum := uint32(0); seqNum < uint32(len(conv.receiver.incoming[pckt.Header.PacketNum])); seqNum++ {
							// Combine Segments
							conv.bskt.basket[pckt.Header.PacketNum] = append(conv.bskt.basket[pckt.Header.PacketNum], conv.receiver.incoming[pckt.Header.PacketNum][seqNum].Body...)
						}
					}

					// Unlock Basket
					//conv.bskt.basket_lock.Unlock()
				}
			}

			// Unlock Receiver
			conv.receiver.incoming_lock.Unlock()
		}

	case ACK:
		{
			fmt.Printf("Got an ACK\n")

			// Lock Sender
			conv.sender.outgoing_lock.Lock()

			// Make sure outgoing packet exists
			if _, exists := conv.sender.outgoing[pckt.Header.PacketNum][pckt.Header.SequenceNum]; !exists {
				log.Printf("Packet %d does not exist, cannot Ack.", pckt.Header.SequenceNum)
				return // Drop Ack
			}

			// Set Ack received state to true
			conv.sender.outgoing[pckt.Header.PacketNum][pckt.Header.SequenceNum].AckReceived = true

			// Unlock Sender
			conv.sender.outgoing_lock.Unlock()
		}

	case NAK:
		{
			fmt.Printf("Got a NACK\n")

			// Lock Sender
			conv.sender.outgoing_lock.Lock()

			// Make sure outgoing packet exists
			if _, exists := conv.sender.outgoing[pckt.Header.PacketNum][pckt.Header.SequenceNum]; !exists {
				log.Printf("Packet %d does not exist, cannot resend for Nack.", pckt.Header.SequenceNum)
				return // Drop Nack
			}

			// Make sure packet wasn't Acked before the lock
			if conv.sender.outgoing[pckt.Header.PacketNum][pckt.Header.SequenceNum].AckReceived == true {
				log.Printf("Packet %d already Acked, won't resend.", pckt.Header.SequenceNum)
				return // Drop Nack
			}

			// Resend Packet
			conv.sendPacket(conv.sender.outgoing[pckt.Header.PacketNum][pckt.Header.SequenceNum])

			// Unlock Sender
			conv.sender.outgoing_lock.Unlock()
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
	fmt.Println("\nSending Hello\n")
	// Create the Hello Struct for the body of the Packet
	helloBody := PcktHello{
		DataID:      hello_c2s,
		Version:     0,
		NumFeatures: 300,
		Features:    []uint16{1, 1, 2, 3, 2, 1, 2, 1, 2, 3, 4, 1, 4, 4},
	}

	helloBody_bytes, err := SerializeHello(&helloBody)
	if err != nil {
		return
	}

	// Lock outgoing
	conv.sender.outgoing_lock.Lock()

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

	conv.fragmentator(&helloPacket)

	// Increment next Packet Number
	conv.sender.nextPcktNum += 1

	conv.sender.outgoing_lock.Unlock()
}

// sendHelloBack sends a Hello Back Packet
func (conv *conversation) sendHelloResonse() {
	// Create the Hello Struct for the body of the Packet
	helloResponseBody := PcktHello{
		DataID:      hello_back_s2c,
		Version:     0,
		NumFeatures: 1,
		Features:    []uint16{1},
	}

	helloResponseBody_bytes, err := SerializeHello(&helloResponseBody)
	if err != nil {
		return
	}

	// Lock outgoing
	conv.sender.outgoing_lock.Lock()

	helloResponsePacket := Pckt{
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

	helloResponsePacket.Body = append(helloResponsePacket.Body, helloResponseBody_bytes...)

	conv.fragmentator(&helloResponsePacket)

	// Increment next Packet Number
	conv.sender.nextPcktNum += 1

	conv.sender.outgoing_lock.Unlock()
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

func (conv *conversation) fragmentator(pckt *Pckt) {

	// Check if it already exists
	if _, exists := conv.sender.outgoing[pckt.Header.PacketNum]; !exists {
		conv.sender.outgoing[pckt.Header.PacketNum] = make(map[uint32]*Pckt)
	}

	// Check if the packet needs to be fragmented
	for seqNum := uint32(0); seqNum < uint32((len(pckt.Body)/FRAGMENT_CONST)+1); seqNum++ {

		// Check if final
		if seqNum+1 >= uint32((len(pckt.Body)/FRAGMENT_CONST)+1) {
			var newPckt Pckt
			newPckt = Pckt{
				Header: PcktHeader{
					Magic:       MAGIC_CONST,
					Checksum:    0,
					ConvID:      pckt.Header.ConvID,
					PacketNum:   pckt.Header.PacketNum,
					SequenceNum: seqNum,
					IsFinal:     1,
					Type:        pckt.Header.Type,
				},
				Body:        make([]byte, 0),
				AckReceived: false,
				LastSent:    time.Now(),
			}

			newPckt.Body = append(newPckt.Body, pckt.Body[seqNum*FRAGMENT_CONST:]...)

			fmt.Println(newPckt)

			// Add to Sliding window
			conv.sender.outgoing[newPckt.Header.PacketNum][newPckt.Header.SequenceNum] = &newPckt
			conv.sender.window = append(conv.sender.window, &newPckt)
		} else {
			var newPckt Pckt
			newPckt = Pckt{
				Header: PcktHeader{
					Magic:       MAGIC_CONST,
					Checksum:    0,
					ConvID:      pckt.Header.ConvID,
					PacketNum:   pckt.Header.PacketNum,
					SequenceNum: seqNum,
					IsFinal:     0,
					Type:        pckt.Header.Type,
				},
				Body:        make([]byte, 0),
				AckReceived: false,
				LastSent:    time.Now(),
			}

			newPckt.Body = append(newPckt.Body, pckt.Body[seqNum*FRAGMENT_CONST:FRAGMENT_CONST+seqNum*FRAGMENT_CONST]...)

			fmt.Println(newPckt)

			// Add to Sliding window
			conv.sender.outgoing[newPckt.Header.PacketNum][newPckt.Header.SequenceNum] = &newPckt
			conv.sender.window = append(conv.sender.window, &newPckt)
		}

		//fmt.Print(conv.sender.window)

		//fmt.Printf("Adding packet: ConversationId=%d, PacketNumber=%d, SequenceNumber=%d, Data=%s, IsFinal=%t\n", newPckt.Header.ConvID, newPckt.Header.SequenceNum, newPckt.Header.SequenceNum, string(newPckt.Body), newPckt.Header.IsFinal != 0)
	}

	fmt.Print(conv.sender.window)
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
	//for range conv.sender.transmission_ticker.C {
	conv.sender.outgoing_lock.Lock()

	conv.moveWindow()

	for i := conv.sender.windowStart; i < conv.sender.windowStart+conv.sender.windowSize; i++ {
		// Make sure we are not going outside of the Slice
		if i < uint32(len(conv.sender.window)) {
			// Make sure it's not a NULL pointer
			if conv.sender.window[i] != nil {
				// Make sure we haven't received an ACK for it before this function
				if !conv.sender.window[i].AckReceived {
					conv.sendPacket(conv.sender.window[i])
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
	//}
}

// Handles the window sliding as packets are acknowledged.
func (conv *conversation) moveWindow() {
	//conv.sender.outgoing_lock.Lock() // Ensure thread-safe access to the sender

	//fmt.Println("Moving window")

	var original_windowStart uint32 = conv.sender.windowStart

	for i := original_windowStart; i < original_windowStart+conv.sender.windowSize; i++ {
		// Make sure we are not going outside of the Slice
		if i < uint32(len(conv.sender.window)) {
			// Make sure it's not a NULL pointer
			if conv.sender.window[i] != nil {
				if conv.sender.window[i].AckReceived {
					// Move window if this packet is ACKed
					conv.sender.windowStart += 1
					fmt.Printf("Packet %d: %d Acked, moved window start to %d\n", conv.sender.window[i].Header.PacketNum, conv.sender.window[i].Header.SequenceNum, conv.sender.windowStart)
				} else {
					// Stop moving window
					break
				}
			} else {
				log.Printf("NULL pointer in window slice")
			}
		} else {
			//log.Printf("Move Window: Reached the end of the window")
			break
		}
	}

	// Remove acknowledged packet state to save memory
	if conv.sender.windowStart > 0 {
		conv.sender.window = conv.sender.window[conv.sender.windowStart:]

		// Reset other variables
		conv.sender.windowStart = 0
	}

	//fmt.Println("Finished Moving Window")

	//conv.sender.outgoing_lock.Unlock() // Ensure thread-safe access to the sender
}

// Implementation of timers for managing packet resends, periodically checks the sliding window and
// resends any packets that have not been acknowledged after a certain time interval.
func (conv *conversation) checkForRetransmissions() {
	//for range conv.sender.retransmission_ticker.C {
	conv.sender.outgoing_lock.Lock() // Ensure thread-safe access to sender

	//fmt.Printf("Window Size %d\n", len(conv.sender.window))

	if len(conv.sender.window) > 0 {
		for i := uint32(0); i < uint32(len(conv.sender.window)); i++ {
			if i < uint32(len(conv.sender.window)) {
				//fmt.Printf("Window Start %d\n", conv.sender.windowStart)
				//fmt.Printf("i = %d\n", i)
				if conv.sender.window[i] != nil {
					//log.Print(conv.sender.window[i].LastSent)
					//log.Print(conv.sender.window[i].AckReceived)
					if !conv.sender.window[i].AckReceived && time.Since(conv.sender.window[i].LastSent) > 1000*time.Millisecond {
						fmt.Println("Resending %d: %d \n", conv.sender.window[i].Header.PacketNum, conv.sender.window[i].Header.SequenceNum)
						conv.sendPacket(conv.sender.window[i])
					}
				}
			} else {
				//fmt.Printf("Check for Retransmission: Going outside of window\n")
				break
			}
		}
	}

	conv.sender.outgoing_lock.Unlock()
	//}
}

/*
func (conv *conversation) incomingProcessor(raw_data []byte) {
	//for range conv.bskt.processing_ticker.C {

	fmt.Println("We got out 4")
	// Lock Basket

	fmt.Println("We are here, basket size is %d", len(conv.bskt.basket))

	// Check if there are any packets waiting to be processed
	if len(conv.bskt.basket) > 0 {

		// Get minimum Packet Number
		// var minPcktNum uint32 = 0xFFFFFFFF
		// for pcktNum := range conv.bskt.basket {
		// 	if minPcktNum > pcktNum && conv.bskt.done[pcktNum] == false {
		// 		minPcktNum = pcktNum
		// 	}
		// }

		// Check if there is a gap between the last packet processed and the minimum next Packet Number
		// if conv.bskt.lastPcktNum+1 < minPcktNum {
		// 	// Unlock Basket, and wait for the missing packet
		// 	conv.bskt.basket_lock.Unlock()
		// 	return
		// 	//continue
		// }

		var minPcktNum uint32

		for pcktNum := range conv.bskt.basket {
			minPcktNum = pcktNum
			break
		}

		// Make sure it actually exists
		if _, exists := conv.bskt.basket[minPcktNum]; !exists {
			log.Printf("Packet %d does not exist in basket.", minPcktNum)
			// Unlock Basket, and panic
			return
			//continue
		}

		// Process the next packet
		DataID, err := DeserializeDataID(conv.bskt.basket[minPcktNum][:2])
		if err != nil {
			log.Printf("Couldn't get Data ID")
			// Unlock Basket, and panic
			return
			//continue
		}

		conv.bskt.done[minPcktNum] = true

		switch DataID {
		case hello_c2s:
			{
				log.Printf("Got a hello\n")
				hello, err := DeserializeHello(conv.bskt.basket[minPcktNum])
				if err != nil {
					log.Printf("Could't Deserialize Hello Packet")
					// Unlock Basket, and panic
					return
					//continue
				}

				fmt.Println(hello.Version)

				// Send a Hello Back
				conv.sendHello()
			}

		case hello_back_s2c:
			{
				log.Printf("Got a hello back\n")
				hello_back, err := DeserializeHello(conv.bskt.basket[minPcktNum])
				if err != nil {
					log.Printf("Could't Deserialize Hello Back Packet")
					// Unlock Basket, and panic
					return
					//continue
				}

				// Print for testing
				fmt.Printf("\nClient's Features...\n")
				fmt.Print(hello_back.Features, "\n\n")

				// Do nothing
				conv.sendHello()
			}

		case vote_c2s_request_vote:
			{
				vote_request, err := DeserializeVoteRequest(conv.bskt.basket[minPcktNum])
				if err != nil {
					log.Printf("Could't Deserialize Vote Request from Client Packet")
					// Unlock Basket, and panic
					return
					//continue
				}

				fmt.Print(vote_request.Question)

				// As Server, Begin a vote
			}

		case vote_s2c_broadcast_question:
			{
				vote_broadcast_question, err := DeserializeVoteRequest(conv.bskt.basket[minPcktNum])
				if err != nil {
					log.Printf("Could't Deserialize Vote Broadcast Question from Server Packet")
					// Unlock Basket, and panic
					return
					//continue
				}

				fmt.Print(vote_broadcast_question.Question)

				// As a Client, Process Question and send your response back to server
			}

		case vote_c2s_response_to_question:
			{
				vote_response, err := DeserializeVoteResponse(conv.bskt.basket[minPcktNum])
				if err != nil {
					log.Printf("Could't Deserialize Vote Response from Client Packet")
					// Unlock Basket, and panic
					return
					//continue
				}

				fmt.Print(vote_response.Response)

				// As Server, log Client response
			}

		case vote_s2c_broadcast_result:
			{
				vote_broadcast_result, err := DeserializeVoteResponse(conv.bskt.basket[minPcktNum])
				if err != nil {
					log.Printf("Could't Deserialize Vote Broadcast Result from Server Packet")
					// Unlock Basket, and panic
					return
					//continue
				}

				fmt.Print(vote_broadcast_result.Response)

				// As Client, overwrite your own response to the Question if you got it wrong
			}

		default:
			{
				log.Printf("Received an Unknown Data ID")
				// Unlock Basket, and panic
				return
				//continue
			}
		}
	}
}
*/

func (conv *conversation) incomingProcessor() {
	//for range conv.bskt.processing_ticker.C {
	// Lock Basket
	conv.receiver.incoming_lock.Lock()

	fmt.Println("We are here, basket size is %d", len(conv.bskt.basket))

	// Check if there are any packets waiting to be processed
	if len(conv.bskt.basket) > 0 {

		fmt.Println("We through")

		// Get minimum Packet Number
		// var minPcktNum uint32 = 0xFFFFFFFF
		// for pcktNum := range conv.bskt.basket {
		// 	if minPcktNum > pcktNum && conv.bskt.done[pcktNum] == false {
		// 		minPcktNum = pcktNum
		// 	}
		// }

		// Check if there is a gap between the last packet processed and the minimum next Packet Number
		// if conv.bskt.lastPcktNum+1 < minPcktNum {
		// 	// Unlock Basket, and wait for the missing packet
		// 	conv.bskt.basket_lock.Unlock()
		// 	return
		// 	//continue
		// }

		var minPcktNum uint32

		for pcktNum := range conv.bskt.basket {
			minPcktNum = pcktNum
			break
		}

		// Make sure it actually exists
		if _, exists := conv.bskt.basket[minPcktNum]; !exists {
			log.Printf("Packet %d does not exist in basket.", minPcktNum)
		} else {
			// Process the next packet
			DataID, err := DeserializeDataID(conv.bskt.basket[minPcktNum][:2])
			if err != nil {
				log.Printf("Couldn't get Data ID")
			} else {
				switch DataID {
				case hello_c2s:
					{
						log.Printf("Got a hello\n")
						hello, err := DeserializeHello(conv.bskt.basket[minPcktNum])
						if err != nil {
							log.Printf("Could't Deserialize Hello Packet")
						} else {
							fmt.Println(hello.Version)

							// Send a Hello Back
							conv.sendHello()
							conv.sendHello()
							conv.sendHello()
							conv.sendHello()
						}
					}

				case hello_back_s2c:
					{
						log.Printf("Got a hello back\n")
						hello_back, err := DeserializeHello(conv.bskt.basket[minPcktNum])
						if err != nil {
							log.Printf("Could't Deserialize Hello Back Packet")
						} else {
							// Print for testing
							fmt.Printf("\nClient's Features...\n")
							fmt.Print(hello_back.Features, "\n\n")

							// Do nothing
							conv.sendHello()
						}
					}

				case vote_c2s_request_vote:
					{
						vote_request, err := DeserializeVoteRequest(conv.bskt.basket[minPcktNum])
						if err != nil {
							log.Printf("Could't Deserialize Vote Request from Client Packet")
						} else {
							fmt.Print(vote_request.Question)

							// As Server, Begin a vote
						}
					}

				case vote_s2c_broadcast_question:
					{
						vote_broadcast_question, err := DeserializeVoteRequest(conv.bskt.basket[minPcktNum])
						if err != nil {
							log.Printf("Could't Deserialize Vote Broadcast Question from Server Packet")
						} else {
							fmt.Print(vote_broadcast_question.Question)

							// As a Client, Process Question and send your response back to server
						}
					}

				case vote_c2s_response_to_question:
					{
						vote_response, err := DeserializeVoteResponse(conv.bskt.basket[minPcktNum])
						if err != nil {
							log.Printf("Could't Deserialize Vote Response from Client Packet")
						} else {
							fmt.Print(vote_response.Response)

							// As Server, log Client response
						}
					}

				case vote_s2c_broadcast_result:
					{
						vote_broadcast_result, err := DeserializeVoteResponse(conv.bskt.basket[minPcktNum])
						if err != nil {
							log.Printf("Could't Deserialize Vote Broadcast Result from Server Packet")
						} else {
							fmt.Print(vote_broadcast_result.Response)

							// As Client, overwrite your own response to the Question if you got it wrong
						}
					}

				default:
					{
						log.Printf("Received an Unknown Data ID")
					}
				}
			}

			// Delete Packet from basket
			delete(conv.bskt.basket, minPcktNum)
		}
	}

	// Unlock Basket
	conv.receiver.incoming_lock.Unlock()

	fmt.Println("and we're out")
	//}
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
