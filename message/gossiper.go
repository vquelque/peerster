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

//PeerStatus gives the next message ID to be received by a particular peer.
type PeerStatus struct {
	Identifier string
	NextID     uint32
}

//StatusMessage is the type of message sent by a particular peer when receiving a message used
// to acknowledge all the messages it has received so far.
type StatusMessage struct {
	Want []PeerStatus
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

//Prints simpleMessage.
func (msg *SimpleMessage) String() string {
	return fmt.Sprintf("SIMPLE MESSAGE origin %s from %s contents %s \n", msg.OriginalName,
		msg.RelayPeerAddr, msg.Contents)
}

//Prints a RumorMessage
func (msg *RumorMessage) String() string {
	return fmt.Sprintf("RUMOR MESSAGE origin %s with ID %s contents %s \n", msg.Origin, msg.ID, msg.Text)
}

//Prints a PeerStatus message
func (ps *PeerStatus) String() string {
	return fmt.Sprintf("Identifier : %s, NextID : %u \n", ps.Identifier, ps.NextID)
}

//Prints a StatusMessage
func (msg *StatusMessage) String() string {
	str := fmt.Sprintf("STATUS MESSAGE received \n")
	for _, element := range msg.Want {
		str += fmt.Sprintf("%s \n", element.String())
	}
	return str
}
