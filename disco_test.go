package disco

import (
	"net"
	"reflect"
	"testing"
	"time"
)

const waitTime = 100 * time.Millisecond

func Test_srvc_String(t *testing.T) {
	type fields struct {
		typ     string
		srcAddr string
		name    string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{"Valid announce", fields{typ: TypeAnnounce, srcAddr: "192.168.0.1:1234", name: "hello"}, "srvc;announce;192.168.0.1:1234;hello"},
		{"Valid query", fields{typ: TypeQuery, srcAddr: "192.168.0.1:1234", name: "hello"}, "srvc;query;192.168.0.1:1234;hello"},
		{"Valid response", fields{typ: TypeResponse, srcAddr: "192.168.0.1:1234", name: "hello"}, "srvc;response;192.168.0.1:1234;hello"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := srvc{
				typ:     tt.fields.typ,
				srcAddr: tt.fields.srcAddr,
				name:    tt.fields.name,
			}
			if got := s.String(); got != tt.want {
				t.Errorf("srvc.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_srvcFrom(t *testing.T) {
	type args struct {
		msg string
	}
	tests := []struct {
		name    string
		args    args
		want    srvc
		wantErr bool
	}{
		{"Valid Announce", args{"srvc;announce;192.168.0.1:1234;hello"}, srvc{typ: TypeAnnounce, srcAddr: "192.168.0.1:1234", name: "hello"}, false},
		{"Valid Query", args{"srvc;query;192.168.0.1:1234;somename"}, srvc{typ: TypeQuery, srcAddr: "192.168.0.1:1234", name: "somename"}, false},
		{"Valid Response", args{"srvc;response;192.168.0.1:1234;somename"}, srvc{typ: TypeResponse, srcAddr: "192.168.0.1:1234", name: "somename"}, false},

		{"Missing Protocol", args{"query;192.168.0.1:1234;hello"}, srvc{}, true},
		{"Missing Protocol with semicolon", args{";query;192.168.0.1:1234;hello"}, srvc{}, true},
		{"Missing Type", args{"srvc;192.168.0.1:1234;hello"}, srvc{}, true},
		{"Missing Type with semicolon", args{"srvc;;192.168.0.1:1234;hello"}, srvc{}, true},

		{"Missing Address", args{"srvc;announce;hello"}, srvc{}, true},
		{"Missing Address with semicolon", args{"srvc;announce;;hello"}, srvc{}, true},

		{"Missing Name", args{"srvc;192.168.0.1:1234;announce"}, srvc{}, true},
		{"Missing Name with semicolon", args{"srvc;announce;192.168.0.1:1234;"}, srvc{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := srvcFrom(tt.args.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("srvcFrom() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("srvcFrom() = %v, want %v", got, tt.want)
			}
		})
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

			service, _ := srvcFrom(msg.Message)

			if service.name != tt.args.name {
				t.Errorf("Announced name %q, got %q", tt.args.name, service.name)
				return
			}

			if service.srcAddr != tt.args.srcAddr {
				t.Errorf("Announced srcAddr %q, got %q", tt.args.srcAddr, service.srcAddr)
				return
			}

			go func() {
				time.Sleep(waitTime)
				Broadcast(tt.args.mAddr, srvc{typ: TypeQuery, srcAddr: "192.168.0.1:1234", name: tt.args.name}.String())
			}()

			// Catch and discard the broadcast
			<-c

			// Catch the response
			msg = <-c

			service, _ = srvcFrom(msg.Message)

			if service.typ != TypeResponse {
				t.Errorf("Service did not respond to query")
				return
			}

			if service.name != tt.args.name {
				t.Errorf("Response name %q, got %q", tt.args.name, service.name)
				return
			}

			if service.srcAddr != tt.args.srcAddr {
				t.Errorf("Response srcAddr %q, got %q", tt.args.srcAddr, service.srcAddr)
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
