package message

import "fmt"

//Message corresponds to the message send from the UI client to gossiper
type Message struct {
	Text        string
	Destination string
	File        string
	Request     []byte
}

//Print client message
func (msg *Message) String() string {
	str := "CLIENT MESSAGE "
	if msg.File != "" && msg.Request == nil {
		str += "FILE SHARE " + msg.File
	} else if msg.File != "" && msg.Request != nil {
		str += fmt.Sprintf("FILE REQUEST METAHASH %x", msg.Request)
	} else {
		str += msg.Text
	}
	return str
}
