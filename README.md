# Consensus Protocol

Currently have a conversation structure model working, where rather than using the Sender's IP, a conversation can be maintained by keeping track of the Sender's unique conversation ID. This way conversations between different nodes can be kept on their separate threads, while using a single listening socket/port. 

Everything for now can be found in the `/udp` directory. Where I modified Ken's `Server_UDP_ARQ.go` file, now it's called `listener.go` just to make it more appropriate, to work with the Conversation Structure, also added the *conversation_id* parameter to the `packet.go` file to make this work.

To test this, use the following commands from within the `udp/` directory:
 - go run listener.go conversation.go packet.go
 - go run client_UDP_ARQ.go packet.go

I encourage you to launch as many windows of the client as you want at the same time, I modified the client so that it generates a random conversation ID so you can keep track of who is calling the server. But this should demonstrate the ability for the server to maintain multiple conversations at the same time. Bear in mind these conversations can be later used to resume transmission.