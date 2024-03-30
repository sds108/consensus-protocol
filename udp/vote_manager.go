// Defines the Vote Manager Structure
package main

import (
	"github.com/google/uuid"
)

// This is hosted on the server
type host_referendum struct {
	// Info
	VoteID         uuid.UUID
	QuestionLength uint32
	Question       string
	ongoing        bool // This tells the program if the Vote is still going

	// Participants, this is a copy of the conversations map at the time of making the vote,
	// the index is the conversationID of each participant
	participants map[uint32]*conversation

	// Participant Responses, the index is the conversationID of each participant
	votes map[uint32]uint16
}

// This is the client copy of the referendum to keep track of the final answer
type client_referendum struct {
	// Info
	VoteID         uuid.UUID
	QuestionLength uint32
	Question       string

	// At first, this is the client's own response,
	// then is overwritten by the server broadcast if it was wrong
	result uint16
}

type referendum_manager struct {
	// All referendums
}
