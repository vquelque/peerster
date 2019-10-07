package socket

import (
	"log"
	"net"

	"github.com/vquelque/Peerster/utils"
)

const maxBufferSize = 1024

//Socket is a generic interface representing a socket
type Socket interface {
	Send(data []byte, address string)
	Receive() ([]byte, string)
	Close()
}

//UDPSocket implements the socket inteface and represents a UDP socket
type UDPSocket struct {
	address    *net.UDPAddr
	connection *net.UDPConn
}

// NewUDPSocket creates a new UDP socket.
func NewUDPSocket(addr string) *UDPSocket {
	udpAddr := utils.ToUDPAddr(addr)
	udpConn, err := net.ListenUDP("udp4", udpAddr)
	if err != nil {
		log.Fatal("error creating udp connection")
	}
	s := &UDPSocket{address: udpAddr, connection: udpConn}
	return s
}

//Send data via the given socket
func (socket *UDPSocket) Send(data []byte, addr string) {
	udpAddr := utils.ToUDPAddr(addr)
	if udpAddr != nil {
		socket.connection.WriteTo(data, udpAddr)
	}
}

//Receive data from the given socket
func (socket *UDPSocket) Receive() ([]byte, string) {
	buf := make([]byte, maxBufferSize)
	bytesRead, source, err := socket.connection.ReadFromUDP(buf)
	if err != nil {
		log.Print(err)
	}
	return buf[:bytesRead], source.String()
}

// Close the connection associated to the socket
func (socket *UDPSocket) Close() {
	socket.connection.Close()
}
