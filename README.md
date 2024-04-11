# Consensus Protocol

Functioning Client-Server Consensus Model, where a Static IP Host (or Server) node can welcome user operated "Client" nodes to partake in voting (the scalability of this model allows nodes to vote on anything in theory).

Currently, the only type of voting supported by the Vote Manager (Not the actual protocol itself, the protocol itself can handle almost anything) is Simple Mathematic Expression Verification.

E.g. A "Client" node is able to propose a question to the system like **_"2 * 2 * 2 * 2 == 16"_** to which in theory, most nodes should respond to with a _**"1" (True)**_. 

Consensus works as follows, as the host of a referendum (in this model a Server) receives ballots from "Client" nodes corresponding to said referendum, it checks whether enough ballots have been cast to call an early winner (e.g. if 60% of the connected nodes vote "Yes" to a proposed question, the server does not have to wait for the remaining 40% of nodes to vote, as their votes will not be able to sway the vote another way regardless of how they vote). Once this statement has been satisfied (or if everyone has voted if the vote was a close tie), the host (Server) distributes the verdict made by the system of nodes, back to the nodes themselves. This way, nodes who voted against the system are forced back in line (enforcing consensus on the system) and are forced to overwrite their own computed answer. And the process repeats for each coming question.

While this a very simple use case of the system, computing and comparing simple math expressions, the potential of the protocol itself is quite vast and quite scalable (Imagine using this in a network of AI operated nodes, where nodes teach each other things, e.g. consensus on Image recognition, or Large Language Model training (LLM AI nodes answer each other's language based questions), or even simply high-precision high-TFLOP GPU machines calculating irrational/transcendental numbers comparing answers and gaining consensus on the most accepted values within the scientific/mathematical community).

This is not to say this protocol is complete, it is missing multi-fragment packets (i.e. missing large file transfer), congestion control (However, this would be an easy fix if given more time to work on this project), E2EE, storing session data to files (For timed out nodes, for now inactive connections remain in memory indefinitely), P2P functionality (though this could be made possible with a simple addition of maybe 20 lines), and I'm sure a couple other things are missing as well.

#### Updates:
---
 - Client-side CLI (Command Line Interface)
 - Decoders and Encoders for Packet Header, and types of Packets, in `packet.go` and `pip.go`
 - Checksum and Magic Verification
 - Server Type Node Boot up Sequence in `server.go`
 - Client Type Node Boot up Sequence in `client.go`
 - Selective Repeat Implementation
 - Action Handler for types of Packets
 - Action Methods and Functions
 - Vote manager and handler
 - Ping for Conversation ID generation (client request to server)
 - Stashing unresponsive/taken to be offline client nodes into a standby state (excluding them from referendums)
---
