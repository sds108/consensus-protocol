// This file includes all of the global variables necessary for the function of the Node

// Version 1.3
package main

import (
	"sync"
)

// Global Constants
const MAGIC_CONST = 0x01051117

// Returns MAGIC_CONST in Big Endian Bytes
func MAGIC_BYTES_CONST() []byte {
	return []byte{byte(1), byte(5), byte(11), byte(17)}
}

const SERVER_PORT_CONST = "8080"
const CLIENT_PORT_CONST = "42069"

// Global Objects
var conversations_map_lock sync.Mutex
var conversations_map map[uint32]*conversation
