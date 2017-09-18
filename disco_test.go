package disco

import (
	"net"
	"reflect"
	"strings"
	"testing"
	"time"
)

const waitTime = 300 * time.Millisecond

func TestSRVCToString(t *testing.T) {
	tests := []struct {
		have srvc
		want string
	}{
		{srvc{srcAddr: "192.168.221.1:1234", name: "hello"}, "srvc;192.168.221.1:1234;hello"},
		{srvc{srcAddr: "1.1.1.1:4321", name: "rlog"}, "srvc;1.1.1.1:4321;rlog"},
	}

	for _, test := range tests {
		if test.have.String() != test.want {
			t.Errorf("stringer failed. Wanted %q, got %q", test.want, test.have.String())
		}
	}

}

func TestStringToSRVC(t *testing.T) {
	tests := []struct {
		str   []string
		want  srvc
		exErr bool
	}{
		{[]string{"srvc", "192.168.211.88:1234", "hello"}, srvc{srcAddr: "192.168.211.88:1234", name: "hello"}, false},
		{[]string{"srvc", "1.1.1.1:1234", "rlog"}, srvc{srcAddr: "1.1.1.1:1234", name: "rlog"}, false},
		{[]string{"srvc", "", ""}, srvc{}, true},
		{[]string{"srvc", "192.168.211.1:4569", ""}, srvc{}, true},
		{[]string{"srvc", "", "something"}, srvc{}, true},
		{[]string{"", "192.168.221.1:1234", "hello"}, srvc{}, true},
		{[]string{"192.168.221.1:1234", "hello"}, srvc{}, true},
		{[]string{"asd", "srvc", "192.168.221.1:1234", "hello"}, srvc{}, true},
		{[]string{"asd", "srvc", "192.168.221.1:1234"}, srvc{}, true},
	}

	for _, test := range tests {
		testString := strings.Join(test.str, ";")
		srvc, err := newSRVC(testString)
		if err != nil {
			if test.exErr {
				continue // We're expecting an error, so it's good.
			}
			t.Errorf("newSRVC(%q) error'd: %v", test.str, err)
		}

		if srvc.srcAddr != test.want.srcAddr {
			t.Errorf("newSRVC(%q) address mismatch, got %q", testString, srvc.srcAddr)
		}

		if srvc.name != test.want.name {
			t.Errorf("newSRVC(%q) name mismatch, got %q", testString, srvc.name)
		}
	}
}

func TestSubscribe(t *testing.T) {
	type args struct {
		addr string
	}
	tests := []struct {
		name    string
		args    args
		msg     string
		wantErr bool
	}{
		{"Valid subscribe", args{"224.0.0.1:9999"}, "hello", false},

		{"Missing port", args{"224.0.0.1"}, "hello", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Subscribe(tt.args.addr)
			if (err != nil) != tt.wantErr {
				t.Errorf("Subscribe() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			go func() {
				time.Sleep(waitTime)
				Broadcast(tt.args.addr, tt.msg)
			}()

			msg := <-got

			if msg.Message != tt.msg {
				t.Errorf("Subscribe() got msg = %v, wanted = %v", msg, tt.msg)
			}
		})
	}
}

func Test_resolve(t *testing.T) {
	type args struct {
		addr string
	}
	tests := []struct {
		name    string
		args    args
		want    *net.UDPAddr
		wantErr bool
	}{
		{"Port and address", args{"192.168.0.1:1234"}, &net.UDPAddr{IP: net.IPv4(192, 168, 0, 1), Port: 1234}, false},
		{"No port", args{"192.168.0.1:"}, &net.UDPAddr{IP: net.IPv4(192, 168, 0, 1), Port: 0}, false},

		{"No port no colon", args{"192.168.0.1"}, nil, true},
		{"Port out of range", args{"192.168.0.1:70000"}, nil, true},
		{"Address out of range", args{"256.0.0.1:1234"}, nil, true},
		{"No address", args{":1234"}, &net.UDPAddr{Port: 1234}, false},
		{"Only colon", args{":"}, &net.UDPAddr{Port: 0}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolve(tt.args.addr)
			if (err != nil) != tt.wantErr {
				t.Errorf("resolve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("resolve() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAnnounce(t *testing.T) {
	type args struct {
		mAddr   string
		srcAddr string
		name    string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"Valid announce", args{"224.0.0.1:9999", "192.168.0.1", "service_name"}, false},
		{"Announce no name", args{"224.0.0.1:9999", "192.168.0.1", ""}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Announce(tt.args.mAddr, tt.args.srcAddr, tt.args.name); (err != nil) != tt.wantErr {
				t.Errorf("Announce() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			go func() {
				time.Sleep(waitTime)
				Announce(tt.args.mAddr, tt.args.srcAddr, tt.args.name)
			}()

			c, err := Subscribe(tt.args.mAddr)
			if err != nil {
				t.Errorf("Subscribe() error = %v", err)
				return
			}

			msg := <-c

			srvc, _ := newSRVC(msg.Message)

			if srvc.name != tt.args.name {
				t.Errorf("Announced name %q, got %q", tt.args.name, srvc.name)
				return
			}

			if srvc.srcAddr != tt.args.srcAddr {
				t.Errorf("Announced srcAddr %q, got %q", tt.args.srcAddr, srvc.srcAddr)
				return
			}
		})
	}
}

func TestListenFor(t *testing.T) {
	type args struct {
		addr  string
		names []string
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]bool
		wantErr bool
	}{
		{"One Wait", args{"224.0.0.1:9999", []string{"hello"}}, map[string]bool{"hello": true}, false},
		{"Two Wait", args{"224.0.0.1:9999", []string{"hello", "bello"}}, map[string]bool{"hello": true, "bello": true}, false},
		{"Zero Wait", args{"224.0.0.1:9999", []string{}}, make(map[string]bool), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ListenFor(tt.args.addr, tt.args.names...)
			if (err != nil) != tt.wantErr {
				t.Errorf("ListenFor() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			go func() {
				time.Sleep(waitTime)
				for _, name := range tt.args.names {
					Announce(tt.args.addr, "192.168.0.1:1234", name)
				}
			}()

			for range tt.args.names {
				name := <-got

				if _, ok := tt.want[name]; !ok {
					t.Errorf("Received non-existing value: %q", name)
					return
				}

				delete(tt.want, name)
			}

			if len(tt.want) != 0 {
				t.Errorf("Entries left in map: %#v", tt.want)
			}
		})
	}
}
