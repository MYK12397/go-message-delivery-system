package main

import (
	"fmt"
	"net"
)

const (
	HOST = "127.0.0.1"
	PORT = "8080"
)

func main() {
	hub := NewHub()
	listener, err := net.Listen("tcp", ":"+PORT)
	if err != nil {
		fmt.Printf("error listening server %v\n", err)
		return
	}

	defer listener.Close()
	fmt.Println("Server listening on :8080")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("error accepting connection")
			return
		}
		go hub.handleConnection(conn)
	}
}
