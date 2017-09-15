package disco

import (
	"strings"
	"testing"
)

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
