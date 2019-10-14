package utils

import (
	"log"
	"net"
)

// MapToUDP converts the given array of string addresses to an array of UDP addresses.
func MapToUDP(vs []string) *[]net.UDPAddr {
	vsm := make([]net.UDPAddr, len(vs))
	for i, v := range vs {
		vsm[i] = *ToUDPAddr(v)
	}
	return &vsm
}

// ToUDPAddr converts the string formated address to a UDPAddress.
func ToUDPAddr(addr string) *net.UDPAddr {
	udpAddr, err := net.ResolveUDPAddr("udp4", addr)
	if err != nil {
		log.Print(err)
		return nil
	}
	return udpAddr
}
