// Defines the Vote Manager Structure
package main

import (
	"log"
	"sync"

	"github.com/google/uuid"
)

// This is hosted on the server
type host_referendum struct {
	// Info
	VoteID   uuid.UUID
	Question string
	ongoing  bool // This tells the program if the Vote is still going

	// Mutex lock for this referendum
	referendum_lock sync.Mutex

	// Participants, this is a copy of the conversations map at the time of making the vote,
	// the index is the conversationID of each participant
	participants map[uint32]*conversation

	// Participant Responses, the index is the conversationID of each participant
	votes map[uint32]uint16
}

// This function copies the current conversations we have a connection with into the participants map
// this way if a new client/conversation arrives, it won't mess with this referendum
func (h_referendum *host_referendum) copyConversationsMap() {
	conversations_lock.Lock()
	h_referendum.referendum_lock.Lock()
	for key, conversation_ref := range conversations {
		h_referendum.participants[key] = conversation_ref
	}
	h_referendum.referendum_lock.Unlock()
	conversations_lock.Unlock()
}

// This function is basically a constructor for the host_referendum struct
func (manager *referendum_manager) newHostReferendum(pckt PcktVoteRequest) *host_referendum {
	return &host_referendum{
		VoteID:       pckt.VoteID,
		Question:     pckt.Question,
		ongoing:      true,
		participants: make(map[uint32]*conversation),
	}
}

// This is the client copy of the referendum to keep track of the final answer
type client_referendum struct {
	// Info
	VoteID   uuid.UUID
	Question string

	// Mutex lock for this referendum
	referendum_lock sync.Mutex

	// At first, this is the client's own response,
	// then is overwritten by the server broadcast if it was wrong
	result uint16
}

func (manager *referendum_manager) newClientReferendum() {

}

type referendum_manager struct {
	// Hosted Referendums
	h_referendums      map[uuid.UUID]*host_referendum
	h_referendums_lock sync.Mutex

	// Participating as a client Referenums
	c_referendums      map[uuid.UUID]*client_referendum
	c_referendums_lock sync.Mutex
}

// Used by the Packet Processor when the PcktVoteRequest packet comes in
func (manager *referendum_manager) create_referendum_from_client_request(pckt Pckt) {
	// Decode the Body of the TXP Packet
	pcktvoterequest, err := DeserializeVoteRequest(pckt.Body)
	if err != nil {
		log.Printf("Couldn't Deserialize Vote Request\n")
		return
	}

	// Check for duplicate VoteIDs in host_referendum map
	if _, exists := manager.h_referendums[pcktvoterequest.VoteID]; exists {
		log.Printf("Duplicate Vote ID detected\n")
	}

	// Create a Referendum Object that this Node (server) is hosting
	manager.h_referendums_lock.Lock()
	manager.h_referendums[pcktvoterequest.VoteID] = manager.newHostReferendum(*pcktvoterequest)
	manager.h_referendums[pcktvoterequest.VoteID].copyConversationsMap()
	manager.h_referendums_lock.Unlock()

	// Broadcast Referendum Question to clients
}

// Used by the Packet Processor when the PcktVoteRequest packet comes in, once the Host Referendum has been set up
func (manager *referendum_manager) broadcast_referendum_to_participants(voteid uuid.UUID) {
	// Create the broadcast packet to send
	packet

	// Go to each participant's conversation object and send them the Question
	manager.h_referendums_lock.Lock()

}

// Used by the Packet Processor when the PcktVoteRequest packet comes in
func (*referendum_manager) broadcast_Referendum_to_Clients(pcktvotebroadcast Pckt) {

}
