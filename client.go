package main

import (
	"bufio"
	"fmt"
	"net"
	"net/rpc"
	"os"
	"os/signal"
	"syscall"
)

type ClientReceiver struct{}

func (c *ClientReceiver) ReceiveMessage(args *struct{ Username, Text string }, reply *struct{ Success bool }) error {
	fmt.Printf("\r[%s]: %s\n> ", args.Username, args.Text)
	reply.Success = true
	return nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run client.go <username>")
		return
	}
	username := os.Args[1]

	listener, err := net.Listen("tcp", "localhost:0") 
	if err != nil {
		fmt.Println("Error starting client listener:", err)
		return
	}
	defer listener.Close()
	callbackAddr := listener.Addr().String()

	receiver := &ClientReceiver{}
	rpc.Register(receiver)
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go rpc.ServeConn(conn)
		}
	}()

	client, err := rpc.Dial("tcp", "localhost:8080")
	if err != nil {
		fmt.Println("Could not connect to chat server:", err)
		return
	}
	defer client.Close()

	args := struct{ Username, CallbackAddr string }{Username: username, CallbackAddr: callbackAddr}
	var reply struct {
		Success bool
		Message string
	}

	err = client.Call("ChatServer.Join", &args, &reply)
	if err != nil || !reply.Success {
		fmt.Println("Join failed:", reply.Message)
		return
	}

	fmt.Println(reply.Message)
	fmt.Println("Type your messages below (or type '/quit' to exit):")
	fmt.Print("> ")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		var leaveReply struct{ Success bool }
		_ = client.Call("ChatServer.Leave", &struct{ Username string }{Username: username}, &leaveReply)
		fmt.Println("\nDisconnected. Goodbye!")
		os.Exit(0)
	}()

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		text := scanner.Text()
		if text == "" {
			fmt.Print("> ")
			continue
		}

		if text == "/quit" {
			var leaveReply struct{ Success bool }
			client.Call("ChatServer.Leave", &struct{ Username string }{Username: username}, &leaveReply)
			fmt.Println("Exited chat room.")
			break
		}

		msgArgs := struct{ Username, Text string }{Username: username, Text: text}
		var msgReply struct{ Success bool }
		err = client.Call("ChatServer.SendMessage", &msgArgs, &msgReply)
		if err != nil {
			fmt.Println("Error sending message:", err)
		}
		fmt.Print("> ")
	}
}