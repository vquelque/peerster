package message

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"

	"github.com/vquelque/Peerster/constant"
	"github.com/vquelque/Peerster/utils"
)

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

//Wrapper for RumorMessage/TLCMessage
type RumorPacket struct {
	RumorMessage *RumorMessage
	TLCMessage   *TLCMessage
}

//PeerStatus gives the next message ID to be received by a particular peer.
type PeerStatus struct {
	Identifier string
	NextID     uint32
}

//StatusPacket is exchanged between peers to exchange their vector clocks.
type StatusPacket struct {
	Want []PeerStatus
}

//PrivateMessage between 2 peers
type PrivateMessage struct {
	Origin      string
	ID          uint32
	Text        string
	Destination string
	HopLimit    uint32
}

type DataRequest struct {
	Origin      string
	Destination string
	HopLimit    uint32
	HashValue   []byte //hash of chunk or metafile if file request
}

type DataReply struct {
	Origin      string
	Destination string
	HopLimit    uint32
	HashValue   []byte
	Data        []byte
}

type SearchRequest struct {
	Origin   string
	Budget   uint64
	Keywords []string
}

type SearchReply struct {
	Origin      string
	Destination string
	HopLimit    uint32
	Results     []*SearchResult
}

type SearchResult struct {
	FileName     string
	MetafileHash []byte
	ChunkMap     []uint64
	ChunkCount   uint64
}

type TxPublish struct {
	Name        string
	Size        int64 // Size in bytes
	MetafileHah []byte
}

type BlockPublish struct {
	PrevHash    [32]byte //0 for ex3
	Transaction TxPublish
}

type TLCMessage struct {
	Origin      string
	ID          uint32
	Confirmed   int
	TxBlock     BlockPublish
	VectorClock *StatusPacket
	Fitness     float32
}

type TLCAck *PrivateMessage

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

//Prints a PeerStatus message
func (ps *PeerStatus) String() string {
	return fmt.Sprintf("peer %s nextID %d", ps.Identifier, ps.NextID)
}

//StringStatusWithSender prints a StatusMessage with its sender
func (msg *StatusPacket) StringStatusWithSender(sender string) string {
	str := fmt.Sprintf("STATUS from %s \n", sender)
	for _, element := range msg.Want {
		str += fmt.Sprintf("%s \n", element.String())
	}
	return str
}

//NewPrivateMessage creates a new private message for peer dest (dest is peer identifier not address).
// Set hop limit to 0 for default value
func NewPrivateMessage(origin string, text string, destination string, hoplimit uint32) *PrivateMessage {
	if hoplimit == 0 {
		hoplimit = constant.DefaultHopLimit //default hoplimit
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

// Set hop limit to 0 for default value (10)
func NewDataReply(origin string, hoplimit uint32, request *DataRequest, data []byte) *DataReply {
	if hoplimit <= 0 {
		hoplimit = constant.DefaultHopLimit //default hoplimit
	}
	d := &DataReply{
		Origin:      origin,
		Destination: request.Origin,
		HopLimit:    hoplimit,
		HashValue:   request.HashValue,
		Data:        data,
	}
	return d
}

// Set hop limit to 0 for default value (10)
func NewDataRequest(origin string, destination string, hoplimit uint32, hashValue utils.SHA256) *DataRequest {
	if hoplimit <= 0 {
		hoplimit = 10 //default hoplimit
	}
	dr := &DataRequest{
		Origin:      origin,
		Destination: destination,
		HopLimit:    hoplimit,
		HashValue:   hashValue[:],
	}
	return dr
}

func NewSearchRequest(origin string, keywords []string, budget uint64) *SearchRequest {
	return &SearchRequest{Origin: origin, Keywords: keywords, Budget: budget}
}

func NewSearchResult(filename string, metafile []byte, chunkMap []uint64, chunkCount uint64) *SearchResult {
	return &SearchResult{
		FileName:     filename,
		MetafileHash: metafile,
		ChunkMap:     chunkMap,
		ChunkCount:   chunkCount,
	}
}

func NewSearchReply(origin string, destination string, hoplimit uint32, results []*SearchResult) *SearchReply {
	sr := &SearchReply{
		Origin:      origin,
		Destination: destination,
		HopLimit:    hoplimit,
		Results:     results,
	}
	return sr
}

func NewTLCMessage(origin string, id uint32, txBlock *BlockPublish, confirmed int, TLCStatusPkt *StatusPacket, fitness float32) *TLCMessage {
	return &TLCMessage{
		Origin:      origin,
		ID:          id,
		Confirmed:   confirmed,
		TxBlock:     *txBlock,
		Fitness:     fitness,
		VectorClock: TLCStatusPkt,
	}
}

//Prints a RumorMessage
func (msg *RumorMessage) String(relay string) string {
	var text = msg.Text
	return fmt.Sprintf("RUMOR origin %s from %s ID %d contents %s", msg.Origin, relay, msg.ID, text)
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

// func (tx *TxPublish) Hash() utils.SHA256 {
// 	hash := sha256.Sum256([]byte(tx.Name))
// 	return hash
// }

// GetDetails return underlying origin,ID and if rumorPacket is a rumorMsg or a TLCMessage
func (pkt *RumorPacket) GetDetails() (string, uint32, bool) {
	var origin string
	var id uint32
	var rumorMsg bool
	switch {
	case pkt.RumorMessage != nil:
		origin = pkt.RumorMessage.Origin
		id = pkt.RumorMessage.ID
		rumorMsg = true
	case pkt.TLCMessage != nil:
		origin = pkt.TLCMessage.Origin
		id = pkt.TLCMessage.ID
		rumorMsg = false
	}
	return origin, id, rumorMsg
}

func (tlcmsg *TLCMessage) String(origin string) string {
	var str string
	switch tlcmsg.Confirmed {
	case -1:
		str = fmt.Sprintf("UNCONFIRMED GOSSIP origin %s ID %d file name %s size %d metahash %x",
			tlcmsg.Origin, tlcmsg.ID, tlcmsg.TxBlock.Transaction.Name, tlcmsg.TxBlock.Transaction.Size, tlcmsg.TxBlock.Transaction.MetafileHah)
	default:
		str = fmt.Sprintf("CONFIRMED GOSSIP origin %s ID %d file name %s size %d metahash %x",
			tlcmsg.Origin, tlcmsg.Confirmed, tlcmsg.TxBlock.Transaction.Name, tlcmsg.TxBlock.Transaction.Size, tlcmsg.TxBlock.Transaction.MetafileHah)

	}
	return str
}

func (pkt *RumorPacket) String(origin string) string {
	var str string
	switch {
	case pkt.RumorMessage != nil:
		str = pkt.RumorMessage.String(origin)
	case pkt.TLCMessage != nil:
		str = pkt.TLCMessage.String(origin)
	}
	return str
}

func (b *BlockPublish) Hash() (out [32]byte) {
	h := sha256.New()
	h.Write(b.PrevHash[:])
	th := b.Transaction.Hash()
	h.Write(th[:])
	copy(out[:], h.Sum(nil))
	return
}

func (t *TxPublish) Hash() (out [32]byte) {
	h := sha256.New()
	binary.Write(h, binary.LittleEndian,
		uint32(len(t.Name)))
	h.Write([]byte(t.Name))
	h.Write(t.MetafileHah)
	copy(out[:], h.Sum(nil))
	return
}
