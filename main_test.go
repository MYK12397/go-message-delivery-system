package main

import (
	"bytes"
	"encoding/binary"
	"log"
	"net"
	"testing"
	"time"
)

func TestMessageDeliverySystem(t *testing.T) {
	log.SetFlags(log.Ltime | log.Lmicroseconds)
	log.Println("Starting TestMessageDeliverySystem")

	// Start the hub in a goroutine
	hub := NewHub()
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		t.Fatalf("Failed to start listener: %v", err)
	}
	defer listener.Close()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return // stop if the listener is closed
			}
			go hub.handleConnection(conn)
		}
	}()

	// Wait a bit for the server to start
	time.Sleep(100 * time.Millisecond)

	// Create three clients
	client1, err := NewClient("localhost:8080")
	if err != nil {
		t.Fatalf("Failed to create client1: %v", err)
	}
	defer client1.Close()
	log.Printf("Created client1 with ID %d", client1.userID)

	client2, err := NewClient("localhost:8080")
	if err != nil {
		t.Fatalf("Failed to create client2: %v", err)
	}
	defer client2.Close()
	log.Printf("Created client2 with ID %d", client2.userID)

	client3, err := NewClient("localhost:8080")
	if err != nil {
		t.Fatalf("Failed to create client3: %v", err)
	}
	defer client3.Close()
	log.Printf("Created client3 with ID %d", client3.userID)

	// Add a small delay to ensure all clients are registered
	time.Sleep(100 * time.Millisecond)

	// Test ListMessage
	log.Println("Sending ListMessage from client1")
	userIDs, err := client1.SendListMessage()
	if err != nil {
		t.Fatalf("Failed to send ListMessage: %v", err)
	}
	log.Printf("Received userIDs: %v", userIDs)
	if len(userIDs) != 2 {
		t.Errorf("Expected 2 users, got %d", len(userIDs))
	}
	expectedIDs := map[uint64]bool{client2.userID: true, client3.userID: true}
	for _, id := range userIDs {
		if !expectedIDs[id] {
			t.Errorf("Unexpected user ID in list: %d", id)
		}
	}

	// Test RelayMessage
	message := []byte("Hello, World!")
	err = client1.SendRelayMessage([]uint64{client2.userID, client3.userID}, message)
	if err != nil {
		t.Fatalf("Failed to send RelayMessage: %v", err)
	}

	// Check if client2 and client3 received the message
	checkReceivedMessage(t, client2.conn, message)
	checkReceivedMessage(t, client3.conn, message)
}

func checkReceivedMessage(t *testing.T, conn net.Conn, expectedMessage []byte) {
	var messageType MessageType
	err := binary.Read(conn, binary.BigEndian, &messageType)
	if err != nil {
		t.Fatalf("Failed to read message type: %v", err)
	}
	if messageType != RelayMessage {
		t.Fatalf("Expected RelayMessage, got %d", messageType)
	}

	var messageLength uint32
	err = binary.Read(conn, binary.BigEndian, &messageLength)
	if err != nil {
		t.Fatalf("Failed to read message length: %v", err)
	}

	receivedMessage := make([]byte, messageLength)
	_, err = conn.Read(receivedMessage)
	if err != nil {
		t.Fatalf("Failed to read message: %v", err)
	}

	if !bytes.Equal(receivedMessage, expectedMessage) {
		t.Errorf("Expected message %q, got %q", expectedMessage, receivedMessage)
	}
}
