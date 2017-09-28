package disco

import (
	"fmt"
	"log"
	"net"
	"strings"
	"time"
)

// MaxDatagramSize sets the maximum amount of bytes to be read
var MaxDatagramSize = 8192

// Constants that denote the different types of service messages
const (
	// TypeAnnounce should be used to announce presence
	TypeAnnounce = "announce"

	// TypeQuery should be used to ask whether a service with a
	// specific name is present
	TypeQuery = "query"

	// Response should be used to respond to TypeQuery messages
	TypeResponse = "response"

	// Report should be used to ask everyone on the network to
	// report that they are alive
	TypeReport = "report"
)

// Service represence a name-ipaddress:port combination
type Service struct {
	Name string
	Addr string
}

type srvc struct {
	typ     string
	srcAddr string
	name    string
}

func (s srvc) String() string {
	return fmt.Sprintf("srvc;%s;%s;%s", s.typ, s.srcAddr, s.name)
}

func srvcFrom(msg string) (srvc, error) {
	ss := strings.Split(msg, ";")

	if len(ss) != 4 || ss[0] != "srvc" {
		return srvc{}, fmt.Errorf("missing protocol declaration")
	}

	if ss[1] == "" || ss[2] == "" || ss[3] == "" {
		return srvc{}, fmt.Errorf("missing query type, address or name")
	}

	return srvc{typ: ss[1], srcAddr: ss[2], name: ss[3]}, nil

}

// Announce sends out an announcement on the mAddr
// that other clients can listen to. ListenFor can interpret
// these srvc messages
func Announce(mAddr, srcAddr, name string) error {
	if name == "" {
		return fmt.Errorf("announce: empty name is not valid")
	}

	go respondToQueries(mAddr, srcAddr, name)

	return Broadcast(mAddr, srvc{typ: TypeAnnounce, name: name, srcAddr: srcAddr}.String())
}

func respondToQueries(mAddr, srcAddr, name string) {
	msgs, err := Subscribe(mAddr)
	if err != nil {
		log.Printf("Failed to subscribe to %q: %v", mAddr, err)
	}

	for {
		msg := <-msgs
		service, err := srvcFrom(msg.Message)
		if err != nil {
			continue
		}

		if service.typ == TypeReport || (service.typ == TypeQuery && service.name == name) {
			err = Respond(mAddr, srcAddr, name)
			if err != nil {
				log.Printf("respondToQueries: %v", err)
			}
		}
	}
}

// Query sends out a query type broadcast and waits up until timeout
// for a response.
func Query(mAddr, srcAddr, name string, timeout time.Duration) (Service, error) {
	query := srvc{typ: TypeQuery, srcAddr: srcAddr, name: name}

	retry := time.NewTicker(500 * time.Millisecond)
	defer retry.Stop()

	wait := time.After(timeout)

	c, err := ListenFor(mAddr, name)
	if err != nil {
		return Service{}, fmt.Errorf("listenfor: %v", err)
	}

	err = Broadcast(mAddr, query.String())
	if err != nil {
		return Service{}, fmt.Errorf("query: %v", err)
	}

	go func() {
		for range retry.C {
			Broadcast(mAddr, query.String())
		}
	}()

	select {
	case found := <-c:
		return found, nil
	case <-wait:
		return Service{}, fmt.Errorf("RESPONSE_TIMEOUT")
	}
}

// Respond sends a response type broadcast
func Respond(mAddr, srcAddr, name string) error {
	err := Broadcast(mAddr, srvc{typ: TypeResponse, srcAddr: srcAddr, name: name}.String())
	if err != nil {
		return fmt.Errorf("response: %v", err)
	}

	return nil
}

// ListenFor returns a channel that sends a message if any of the
// names that was requested has announced itself on the multicast
// addr. Once announced, the whole message will be returned and then
// removed from the watchlist
func ListenFor(addr string, names ...string) (<-chan Service, error) {
	recv, err := Subscribe(addr)
	if err != nil {
		return nil, err
	}

	send := make(chan Service)
	go listenfor(recv, send, names)

	return send, nil
}

func listenfor(recv <-chan MulticastMsg, send chan<- Service, names []string) {
	mapping := make(map[string]bool)

	for _, name := range names {
		mapping[name] = true
	}

	for {
		msg := <-recv
		srvc, err := srvcFrom(msg.Message)
		if err != nil {
			continue
		}

		if srvc.typ != TypeAnnounce {
			continue
		}

		if _, ok := mapping[srvc.name]; ok {
			send <- Service{Name: srvc.name, Addr: srvc.srcAddr}
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
			Src:     fmt.Sprintf("%s:%d", src.IP, src.Port),
		}
	}
}
