//GLobal
// This file includes all of the global variables necessary for the function of the Node

// Version 1.3
package main

// "sync"

// Types
const (
	DATA = uint16(0)
	ACK  = uint16(1)
	NAK  = uint16(2)
)

// Global Constants
const MAGIC_CONST = 0x01051117

// Returns MAGIC_CONST in Big Endian Bytes
func MAGIC_BYTES_CONST() []byte {
	return []byte{byte(1), byte(5), byte(11), byte(17)}
}

const SERVER_PORT_CONST = "8080"
const CLIENT_PORT_CONST = "42069"

// Own Conversation ID /////////////////////////////////
var conversation_id_self = uint32(0)

// Global Objects
// var conversations_lock sync.Mutex

// var conversations map[uint32]*conversation
