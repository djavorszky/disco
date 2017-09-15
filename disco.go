package disco

import (
	"fmt"
	"net"
)

// MaxDatagramSize sets the maximum amount of bytes to be read
var MaxDatagramSize = 8192

// Broadcast sends a message to the multicast address
// via UDP. The address should be in an "ipaddr:port" fashion
func Broadcast(addr, message string) error {
	udpAddr, err := resolve(addr)
	if err != nil {
		return fmt.Errorf("broadcast: %v", err)
	}

	c, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return fmt.Errorf("broadcast dial %q: %v", addr, err)
	}
	c.Write([]byte(message))
	c.Close()

	return nil
}

// MulticastMsg is used to communicate a message that was
// received on a multicast channel. Contains information
// about the sender as well, or an error if any arose.
type MulticastMsg struct {
	Message string
	Length  int
	Src     string
	Err     error
}

// Subscribe starts listening to a multicast address via
// UDP. The address should be in an "ipaddr:port" fashion.
func Subscribe(addr string) (<-chan MulticastMsg, error) {
	udpAddr, err := resolve(addr)
	if err != nil {
		return nil, fmt.Errorf("subscribe: %v", err)
	}

	c := make(chan MulticastMsg)

	go listen(udpAddr, c)

	return c, nil
}

func resolve(addr string) (*net.UDPAddr, error) {
	a, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("resolve %q: %v", addr, err)
	}

	return a, nil
}

func listen(addr *net.UDPAddr, c chan MulticastMsg) {
	l, err := net.ListenMulticastUDP("udp", nil, addr)
	if err != nil {
		c <- MulticastMsg{Err: fmt.Errorf("listen: %v", err)}
		close(c)
	}
	l.SetReadBuffer(MaxDatagramSize)

	for {
		msg := make([]byte, MaxDatagramSize)
		n, src, err := l.ReadFromUDP(msg)
		if err != nil {
			c <- MulticastMsg{Err: fmt.Errorf("read: %v", err)}
			close(c)
		}

		c <- MulticastMsg{
			Message: string(msg[:n]),
			Length:  n,
			Src:     fmt.Sprintf("%s:%d", src.IP, src.Port),
		}
	}
}
