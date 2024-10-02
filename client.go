package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
)

type Client struct {
	conn   net.Conn
	userID uint64
}

func NewClient(address string) (*Client, error) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}
	client := &Client{conn: conn}
	err = client.sendIdentityMessage()
	if err != nil {
		conn.Close()
		return nil, err
	}

	return client, nil
}

func (c *Client) sendIdentityMessage() error {
	binary.Write(c.conn, binary.BigEndian, IdentityMessage)
	err := binary.Read(c.conn, binary.BigEndian, &c.userID)
	if err != nil {
		return err
	}
	fmt.Printf("assigned user ID: %d\n", c.userID)
	return nil
}

func (c *Client) SendListMessage() ([]uint64, error) {
	log.Printf("Client %d: Sending ListMessage request", c.userID)
	err := binary.Write(c.conn, binary.BigEndian, ListMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to send list message request: %v", err)
	}

	var count uint16
	err = binary.Read(c.conn, binary.BigEndian, &count)
	if err != nil {
		return nil, fmt.Errorf("failed to read user count: %v", err)
	}
	log.Printf("Client %d: Received count of %d users", c.userID, count)

	userIDs := make([]uint64, count)
	for i := range userIDs {
		err = binary.Read(c.conn, binary.BigEndian, &userIDs[i])
		if err != nil {
			return nil, fmt.Errorf("failed to read user ID: %v", err)
		}
		log.Printf("Client %d: Received user ID %d", c.userID, userIDs[i])
	}

	return userIDs, nil
}

func (c *Client) SendRelayMessage(receivers []uint64, message []byte) error {
	if len(receivers) > MaxReceivers {
		return fmt.Errorf("too many receivers: %d (max %d)", len(receivers), MaxReceivers)
	}
	if len(message) > MaxMessageLength {
		return fmt.Errorf("message too long: %d bytes (max %d)", len(message), MaxMessageLength)
	}

	binary.Write(c.conn, binary.BigEndian, RelayMessage)
	binary.Write(c.conn, binary.BigEndian, uint8(len(receivers)))
	for _, receiver := range receivers {
		binary.Write(c.conn, binary.BigEndian, receiver)
	}
	binary.Write(c.conn, binary.BigEndian, uint32(len(message)))
	c.conn.Write(message)

	return nil
}

func (c *Client) ReceiveMessages() {
	for {
		var messageType MessageType
		err := binary.Read(c.conn, binary.BigEndian, &messageType)
		if err != nil {
			if err != io.EOF {
				log.Printf("Error reading message type: %v", err)
			}
			return
		}

		if messageType == RelayMessage {
			var messageLength uint32
			binary.Read(c.conn, binary.BigEndian, &messageLength)

			message := make([]byte, messageLength)
			_, err := io.ReadFull(c.conn, message)
			if err != nil {
				log.Printf("Error reading message: %v", err)
				return
			}

			fmt.Printf("Received message: %s\n", string(message))
		}
	}
}

func (c *Client) Close() error {
	return c.conn.Close()
}
