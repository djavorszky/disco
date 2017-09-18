package disco

import (
	"fmt"
	"net"
	"strings"
)

// MaxDatagramSize sets the maximum amount of bytes to be read
var MaxDatagramSize = 8192

type srvc struct {
	name    string
	srcAddr string
}

func (s srvc) String() string {
	return fmt.Sprintf("srvc;%s;%s", s.srcAddr, s.name)
}

func newSRVC(msg string) (srvc, error) {
	ss := strings.Split(msg, ";")

	if len(ss) != 3 || ss[0] != "srvc" {
		return srvc{}, fmt.Errorf("missing protocol declaration")
	}

	if ss[1] == "" || ss[2] == "" {
		return srvc{}, fmt.Errorf("missing address or name")
	}

	return srvc{srcAddr: ss[1], name: ss[2]}, nil

}

// Announce sends out an announcement on the mAddr
// that other clients can listen to. ListenFor can interpret
// these srvc messages
func Announce(mAddr, srcAddr, name string) error {
	if name == "" {
		return fmt.Errorf("announce: empty name is not valid")
	}

	return Broadcast(mAddr, srvc{name: name, srcAddr: srcAddr}.String())
}

// ListenFor returns a channel that sends a message if any of the
// names that was requested has announced itself on the multicast
// addr. Once announced, the name itself will be returned and then
// removed from the watchlist
func ListenFor(addr string, names ...string) (<-chan string, error) {
	recv, err := Subscribe(addr)
	if err != nil {
		return nil, err
	}

	send := make(chan string)
	go listenfor(recv, send, names)

	return send, nil
}

func listenfor(recv <-chan MulticastMsg, send chan<- string, names []string) {
	mapping := make(map[string]bool)

	for _, name := range names {
		mapping[name] = true
	}

	for {
		msg := <-recv
		srvc, err := newSRVC(msg.Message)
		if err != nil {
			continue
		}

		if _, ok := mapping[srvc.name]; ok {
			send <- srvc.name
			delete(mapping, srvc.name)
		}

		if len(mapping) == 0 {
			close(send)
			return
		}
	}
}

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
