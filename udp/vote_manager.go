// Defines the Vote Manager Structure
package main

import (
	"fmt"
	"log"
	"math/rand"
	"sync"

	"github.com/expr-lang/expr"
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

	// Final Result
	result uint16
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
		result:       SYNTAX_ERROR,
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

	manager.h_referendums_lock.Lock()
	defer manager.h_referendums_lock.Unlock()

	// Check for duplicate VoteIDs in host_referendum map
	if _, exists := manager.h_referendums[pckt.VoteID]; exists {
		log.Printf("Duplicate Vote ID detected\n")
	}

	// Create a Referendum Object that this Node (server) is hosting
	manager.h_referendums[pckt.VoteID] = manager.newHostReferendum(pckt)
	manager.h_referendums[pckt.VoteID].copyConversationsMap()

	// Broadcast Referendum Question to clients
	for _, participant := range manager.h_referendums[pckt.VoteID].participants {
		participant.sendVoteBroadcastToClient(manager.h_referendums[pckt.VoteID])
	}
}

// Used by the Packet Processor when the PcktVoteBroadcast packet comes in on a client
func (manager *referendum_manager) handle_new_question_from_server(pckt *PcktVoteRequest, asker *conversation) {
	fmt.Println("got here 1")
	manager.c_referendums_lock.Lock()
	defer manager.c_referendums_lock.Unlock()

	// Check for duplicate VoteIDs in host_referendum map
	if _, exists := manager.c_referendums[pckt.VoteID]; exists {
		log.Printf("Duplicate Vote ID detected\n")
	}

	// Create a Referendum Object that this Node (server) is hosting
	manager.c_referendums[pckt.VoteID] = manager.newClientReferendum(pckt)

	// Lock referendum
	manager.c_referendums[pckt.VoteID].referendum_lock.Lock()
	defer manager.c_referendums[pckt.VoteID].referendum_lock.Unlock()
	manager.c_referendums[pckt.VoteID].result = manager.computeQuestion(pckt.Question)

	// Send Response back to server (asker, conversation who asked)
	if asker != nil {
		asker.sendResponseToServer(manager.c_referendums[pckt.VoteID])
	}
}

func (manager *referendum_manager) computeQuestion(question string) uint16 {
	var response uint16

	// use evaluate function to return result
	program, err := expr.Compile(question, expr.AsBool())
	if err != nil {
		log.Println("Error compiling question: ", err)
		response = SYNTAX_ERROR
	}

	compute, err := expr.Run(program, nil)
	if err != nil {
		log.Println("Error running question: ", err)
		response = SYNTAX_ERROR
	} else {
		if compute.(bool) {
			response = SAT
		} else {
			response = UNSAT
		}

		// Flip the value by chance
		if rand.Intn(2) == 1 {
			if response == SAT {
				response = UNSAT
			} else if response == UNSAT {
				response = SAT
			}
		}
	}

	fmt.Println("Computed Response: ", response)

	return response
}

func (manager *referendum_manager) handle_response_from_client(pckt *PcktVoteResponse, responder *conversation) {
	manager.h_referendums_lock.Lock()
	defer manager.h_referendums_lock.Unlock()

	// Check if referendum exists
	if _, exists := manager.h_referendums[pckt.VoteID]; !exists {
		log.Printf("Referendum doesn't exist\n")
		return
	}

	// Lock h_referendum
	manager.h_referendums[pckt.VoteID].referendum_lock.Lock()
	defer manager.h_referendums[pckt.VoteID].referendum_lock.Unlock()

	// Check if referendum is still going
	if manager.h_referendums[pckt.VoteID].ongoing == false {
		log.Printf("Referendum is over\n")
		return
	}

	// check if responder (client voting) is not nil
	if responder == nil {
		log.Printf("Responder doesn't exist\n")
		return
	}

	// Check if the responder can vote here
	if _, exists := manager.h_referendums[pckt.VoteID].participants[responder.conversation_id]; !exists {
		log.Printf("Responder is not a legible participant in this vote\n")
		return
	}

	// Check if the responder has already voted
	if _, exists := manager.h_referendums[pckt.VoteID].who[responder.conversation_id]; exists {
		log.Printf("Responder has already voted\n")
		return
	}

	// Add vote now
	manager.h_referendums[pckt.VoteID].who[responder.conversation_id] = true
	if _, exists := manager.h_referendums[pckt.VoteID].votes[pckt.Response]; exists {
		manager.h_referendums[pckt.VoteID].votes[pckt.Response] += 1
	} else {
		manager.h_referendums[pckt.VoteID].votes[pckt.Response] = 1
	}

	// Check if you can broadcast now
	manager.broadcast_result_to_clients(manager.h_referendums[pckt.VoteID])

}

func (manager *referendum_manager) broadcast_result_to_clients(voteRef *host_referendum) {
	// Check requirements to cast a vote
	missingVotes := len(voteRef.participants) - len(voteRef.who)

	// Check if any option has more votes than the remaining votes
	winners := make(map[uint16]bool)
	for r, v := range voteRef.votes {
		if v > uint64(missingVotes) {
			winners[r] = true
			voteRef.result = r
		}
	}

	if len(winners) == 0 {
		fmt.Println("The referendum result cannot be called early")
	} else if len(winners) > 0 {
		fmt.Printf("Option %d has won the referendum\n", winners[0])
		// Go to each participant's conversation object and send them the Question
		for _, participant := range voteRef.participants {
			participant.sendResultBroadcastToClient(voteRef)
		}

		// Call finished vote
		voteRef.ongoing = false
	}
}

func (manager *referendum_manager) handle_result_from_server(pckt *PcktVoteResponse) {
	manager.c_referendums_lock.Lock()
	defer manager.c_referendums_lock.Unlock()

	// Check for VoteID in host_referendum map
	if _, exists := manager.c_referendums[pckt.VoteID]; !exists {
		log.Printf("Could not find referendum mentioned\n")
		return
	}

	// Lock referendum
	manager.c_referendums[pckt.VoteID].referendum_lock.Lock()
	defer manager.c_referendums[pckt.VoteID].referendum_lock.Unlock()

	// Overwrite if necessary
	if manager.c_referendums[pckt.VoteID].result != pckt.Response {
		log.Printf("\nConsensus voted against us for Vote ID: %d, Question: %s.\n Our answer was %d, and the consensus answer was %d, overwriting now.\n\n", pckt.VoteID, manager.c_referendums[pckt.VoteID].Question, manager.c_referendums[pckt.VoteID].result, pckt.Response)
		manager.c_referendums[pckt.VoteID].result = pckt.Response
	} else {
		log.Printf("\nConsensus agrees with us for Vote ID: %d, Question: %s.\n Our answer was %d, and the consensus answer was %d.\n\n", pckt.VoteID, manager.c_referendums[pckt.VoteID].Question, manager.c_referendums[pckt.VoteID].result, pckt.Response)
	}

}
