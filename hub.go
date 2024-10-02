package main

import (
	"encoding/binary"
	"io"
	"log"
	"net"
	"sync"
)

const (
	MaxReceivers     = 255
	MaxMessageLength = 1024 * 1024
)

type MessageType byte

const (
	IdentityMessage MessageType = iota
	ListMessage
	RelayMessage
)

type Hub struct {
	clients    map[uint64]net.Conn
	nextUserID uint64
	mu         sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[uint64]net.Conn),
		nextUserID: 1,
	}
}

func (h *Hub) handleConnection(conn net.Conn) {
	defer conn.Close()

	userID := h.registerClient(conn)
	defer h.unregisterClient(userID)
	for {
		//read incoming message.
		messageType, err := readMessageType(conn)
		if err != nil {
			if err != io.EOF {
				log.Printf("error reading message type: %v", err)
			}
			return
		}

		switch messageType {
		case IdentityMessage:
			h.HandleIdentityMessage(conn, userID)
		case ListMessage:
			h.HandleListMessage(conn, userID)
		case RelayMessage:
			h.HandleRelayMessage(conn)
		default:
			log.Printf("unknown message type: %d", messageType)
		}
	}

}

func (h *Hub) registerClient(conn net.Conn) uint64 {
	h.mu.Lock()
	defer h.mu.Unlock()

	userID := h.nextUserID
	h.clients[userID] = conn
	h.nextUserID++
	log.Printf("Registered client with ID %d. Total clients: %d", userID, len(h.clients))
	return userID
}

func (h *Hub) unregisterClient(userID uint64) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.clients, userID)
}

func (h *Hub) HandleIdentityMessage(conn net.Conn, userID uint64) {
	binary.Write(conn, binary.BigEndian, userID)
}

func (h *Hub) HandleListMessage(conn net.Conn, excludeUserID uint64) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var userIDs []uint64
	for userID := range h.clients {
		if userID != excludeUserID {
			userIDs = append(userIDs, userID)
		}
	}

	log.Printf("Sending list of %d users to client %d", len(userIDs), excludeUserID)

	// Write the count of user IDs
	err := binary.Write(conn, binary.BigEndian, uint16(len(userIDs)))
	if err != nil {
		log.Printf("Error writing user count: %v", err)
		return
	}

	// Write each user ID
	for _, userID := range userIDs {
		err := binary.Write(conn, binary.BigEndian, userID)
		if err != nil {
			log.Printf("Error writing user ID: %v", err)
			return
		}
		log.Printf("Sent user ID %d to client %d", userID, excludeUserID)
	}
}

func (h *Hub) HandleRelayMessage(conn net.Conn) {
	var receiverCount uint8
	binary.Read(conn, binary.BigEndian, &receiverCount)

	receivers := make([]uint64, receiverCount)
	for i := range receivers {
		binary.Read(conn, binary.BigEndian, &receivers[i])
	}

	var messageLength uint32
	binary.Read(conn, binary.BigEndian, &messageLength)

	if messageLength > MaxMessageLength {
		log.Printf("Message too long: %d bytes", messageLength)
		return
	}

	message := make([]byte, messageLength)
	_, err := io.ReadFull(conn, message)
	if err != nil {
		log.Printf("Error reading message: %v", err)
		return
	}

	h.relayMessage(receivers, message)
}

func (h *Hub) relayMessage(receivers []uint64, message []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, receiverID := range receivers {
		if conn, ok := h.clients[receiverID]; ok {
			binary.Write(conn, binary.BigEndian, RelayMessage)
			binary.Write(conn, binary.BigEndian, uint32(len(message)))
			conn.Write(message)
		}
	}
}

func readMessageType(conn net.Conn) (MessageType, error) {
	var messageType MessageType
	err := binary.Read(conn, binary.BigEndian, &messageType)
	return messageType, err
}
