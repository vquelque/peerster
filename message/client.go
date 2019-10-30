package message

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
	if msg.File != "" {
		str += "FILE SHARE " + msg.File
	} else {
		str += msg.Text
	}
	return str
}
