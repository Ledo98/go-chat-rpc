package main

import (
	"fmt"
	"net"
	"net/rpc"
	"sync"
)

type JoinArgs struct {
	Username       string
	CallbackAddr   string 
}

type JoinReply struct {
	Success bool
	Message string
}

type MessageArgs struct {
	Username string
	Text     string
}

type MessageReply struct {
	Success bool
}

type LeaveArgs struct {
	Username string
}

type LeaveReply struct {
	Success bool
}


type ChatServer struct {
	mu      sync.Mutex
	clients map[string]string 
}


func (s *ChatServer) Join(args *JoinArgs, reply *JoinReply) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.clients[args.Username]; exists {
		reply.Success = false
		reply.Message = "Username already taken!"
		return nil
	}

	s.clients[args.Username] = args.CallbackAddr
	reply.Success = true
	reply.Message = "Joined successfully!"

	fmt.Printf("[Server] User joined: %s\n", args.Username)

	
	go s.broadcastMessage("System", fmt.Sprintf("--> %s has joined the chat.", args.Username), args.Username)

	return nil
}

func (s *ChatServer) SendMessage(args *MessageArgs, reply *MessageReply) error {
	fmt.Printf("[%s]: %s\n", args.Username, args.Text)
	go s.broadcastMessage(args.Username, args.Text, args.Username)
	reply.Success = true
	return nil
}

func (s *ChatServer) Leave(args *LeaveArgs, reply *LeaveReply) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.clients, args.Username)
	fmt.Printf("[Server] User left: %s\n", args.Username)

	go s.broadcastMessage("System", fmt.Sprintf("<-- %s has left the chat.", args.Username), args.Username)

	reply.Success = true
	return nil
}

func (s *ChatServer) broadcastMessage(sender, text, excludeUser string) {
	s.mu.Lock()
	clientsCopy := make(map[string]string)
	for u, addr := range s.clients {
		if u != excludeUser {
			clientsCopy[u] = addr
		}
	}
	s.mu.Unlock()

	for _, addr := range clientsCopy {
		go func(clientAddr string) {
			client, err := rpc.Dial("tcp", clientAddr)
			if err != nil {
				return
			}
			defer client.Close()

			var reply MessageReply
			args := MessageArgs{Username: sender, Text: text}
			_ = client.Call("ClientReceiver.ReceiveMessage", &args, &reply)
		}(addr)
	}
}

func main() {
	server := &ChatServer{
		clients: make(map[string]string),
	}

	rpc.Register(server)

	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println("Error starting server:", err)
		return
	}
	defer listener.Close()

	fmt.Println("Chat Server is running on port 8080...")
	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go rpc.ServeConn(conn)
	}
}