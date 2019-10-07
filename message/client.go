package message

import "fmt"

//UIMessage corresponds to the message send from the UI client to gossiper
type UIMessage struct {
	Msg string
}

//Print client message
func (msg *UIMessage) String() string {
	return fmt.Sprintf("CLIENT MESSAGE %s", msg.Msg)
}
