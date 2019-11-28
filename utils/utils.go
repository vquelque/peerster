package utils

import (
	"fmt"
	"io"
	"net"
	"os"
)

type SHA256 = [32]byte

type AdditionalFlags struct {
	HW3ex2          bool
	HW3ex3          bool
	PeersNumber     uint64
	StubbornTimeout int
	AckAll          bool
}

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
		//	log.Print(err)
		return nil
	}
	return udpAddr
}

func CopyFile(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	destination, err := os.Create(dst + sourceFileStat.Name())
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}

func SliceToHash(hash []byte) SHA256 {
	var h SHA256
	copy(h[:], hash)
	return h
}

func ChunkMapToString(cmap []uint64) string {
	var str string
	for i, c := range cmap {
		if i > 0 {
			str += ","
		}
		str += fmt.Sprintf("%d", c)
	}
	return str
}

func Contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
