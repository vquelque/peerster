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
