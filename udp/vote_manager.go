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

	// Who has voted already, the index is the conversationID of each participant
	who map[uint32]bool

	// Each response counter
	votes map[uint16]uint64
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
func (manager *referendum_manager) newHostReferendum(pckt *PcktVoteRequest) *host_referendum {
	return &host_referendum{
		VoteID:       pckt.VoteID,
		Question:     pckt.Question,
		ongoing:      true,
		participants: make(map[uint32]*conversation),
		who:          make(map[uint32]bool),
		votes:        make(map[uint16]uint64),
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

func (manager *referendum_manager) newClientReferendum(pckt *PcktVoteRequest) *client_referendum {
	return &client_referendum{
		VoteID:   pckt.VoteID,
		Question: pckt.Question,
	}
}

type referendum_manager struct {
	// Hosted Referendums
	h_referendums      map[uuid.UUID]*host_referendum
	h_referendums_lock sync.Mutex

	// Participating as a client Referenums
	c_referendums      map[uuid.UUID]*client_referendum
	c_referendums_lock sync.Mutex
}

func newReferendumManager() *referendum_manager {
	return &referendum_manager{
		h_referendums: make(map[uuid.UUID]*host_referendum),
		c_referendums: make(map[uuid.UUID]*client_referendum),
	}
}

// Used by the Packet Processor when the PcktVoteRequest packet comes in
func (manager *referendum_manager) create_referendum_from_client_request(pckt *PcktVoteRequest) {

	// Check for duplicate VoteIDs in host_referendum map
	if _, exists := manager.h_referendums[pckt.VoteID]; exists {
		log.Printf("Duplicate Vote ID detected\n")
	}

	// Create a Referendum Object that this Node (server) is hosting
	manager.h_referendums_lock.Lock()
	defer manager.h_referendums_lock.Unlock()

	manager.h_referendums[pckt.VoteID] = manager.newHostReferendum(pckt)
	manager.h_referendums[pckt.VoteID].copyConversationsMap()

	// Broadcast Referendum Question to clients
	manager.broadcast_referendum_to_participants(manager.h_referendums[pckt.VoteID])
}

// Used by the Packet Processor when the PcktVoteRequest packet comes in, once the Host Referendum has been set up
func (manager *referendum_manager) broadcast_referendum_to_participants(voteRef *host_referendum) {
	// Check requirements to cast a vote

	// Go to each participant's conversation object and send them the Question

}

// Used by the Packet Processor when the PcktVoteBroadcast packet comes in on a client
func (*referendum_manager) handle_new_question_from_server(pcktvotebroadcast *Pckt) {

}
