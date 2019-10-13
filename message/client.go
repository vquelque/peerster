package message

import "fmt"

//Message corresponds to the message send from the UI client to gossiper
type Message struct {
	Msg string
}

//Print client message
func (msg *Message) String() string {
	return fmt.Sprintf("CLIENT MESSAGE %s", msg.Msg)
}
