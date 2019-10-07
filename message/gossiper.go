package message

import "fmt"

// SimpleMessage represents a type of Peerster message containing only text.
type SimpleMessage struct {
	OriginalName  string
	RelayPeerAddr string
	Contents      string
}

func NewSimpleMessage(contents string, originalName string, relayPeerAddr string) *SimpleMessage {
	return &SimpleMessage{
		OriginalName:  originalName,
		RelayPeerAddr: relayPeerAddr,
		Contents:      contents,
	}
}

//Print simple message
func (msg *SimpleMessage) String() string {
	return fmt.Sprintf("SIMPLE MESSAGE origin %s from %s contents %s \n", msg.OriginalName,
		msg.RelayPeerAddr, msg.Contents)
}
