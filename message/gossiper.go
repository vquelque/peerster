package message

import "fmt"

// SimpleMessage represents a type of Peerster message containing only text.
type SimpleMessage struct {
	OriginalName  string
	RelayPeerAddr string
	Contents      string
}

//RumorMessage represents a type of Peerster message to be gossiped.
type RumorMessage struct {
	Origin string
	ID     uint32
	Text   string
}

//PrivateMessage between 2 peers
type PrivateMessage struct {
	Origin      string
	ID          uint32
	Text        string
	Destination string
	HopLimit    uint32
}

//NewSimpleMessage creates a new simpleMessage.
func NewSimpleMessage(contents string, originalName string, relayPeerAddr string) *SimpleMessage {
	return &SimpleMessage{
		OriginalName:  originalName,
		RelayPeerAddr: relayPeerAddr,
		Contents:      contents,
	}
}

//NewRumorMessage creates a new rumorMessage.
func NewRumorMessage(origin string, ID uint32, text string) *RumorMessage {
	return &RumorMessage{
		Origin: origin,
		ID:     ID,
		Text:   text,
	}
}

//NewPrivateMessage creates a new private message for peer dest (dest is peer identifier not address).
// Set hop limit to 0 for default value (10)
func NewPrivateMessage(origin string, text string, destination string, hoplimit uint32) *PrivateMessage {
	if hoplimit <= 0 {
		hoplimit = 10 //default hoplimit
	}
	return &PrivateMessage{
		Origin:      origin,
		ID:          0, //no sequencing for private messages
		Text:        text,
		Destination: destination,
		HopLimit:    hoplimit,
	}
}

// NewRouteRumorMessage creates a route rumor message used to updating routing table
// entries of a peer. It is simply a rumor message with empty text field
func NewRouteRumorMessage(origin string, ID uint32) *RumorMessage {
	return &RumorMessage{
		Origin: origin,
		ID:     ID,
		Text:   "",
	}
}

//Prints a RumorMessage
func (msg *RumorMessage) PrintRumor(relay string) string {
	return fmt.Sprintf("RUMOR origin %s from %s ID %d contents %s", msg.Origin, relay, msg.ID, msg.Text)
}

//Prints simpleMessage.
func (msg *SimpleMessage) String() string {
	return fmt.Sprintf("SIMPLE MESSAGE origin %s from %s contents %s", msg.OriginalName,
		msg.RelayPeerAddr, msg.Contents)
}

// Printes privateMessage
func (msg *PrivateMessage) String() string {
	return fmt.Sprintf("PRIVATE origin %s hop-limit %d contents %s",
		msg.Origin, msg.HopLimit, msg.Text)
}
