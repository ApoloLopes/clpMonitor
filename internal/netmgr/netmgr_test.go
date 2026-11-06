package netmgr

import "testing"

func TestParseCIDR(t *testing.T) {
	ok := map[string]string{"192.168.0.10/24": "192.168.0.10/24", " 10.0.0.1/8 ": "10.0.0.1/8"}
	for in, want := range ok {
		if got, err := parseCIDR(in); err != nil || got != want {
			t.Errorf("parseCIDR(%q) = %q, %v; esperado %q", in, got, err, want)
		}
	}
	for _, in := range []string{"", "abc", "192.168.0.10", "192.168.0.10/33", "::1/128", "1.2.3.4/24; reboot"} {
		if _, err := parseCIDR(in); err == nil {
			t.Errorf("parseCIDR(%q) deveria falhar", in)
		}
	}
}

func TestListHasInterfaces(t *testing.T) {
	info, err := List()
	if err != nil || len(info.Interfaces) == 0 {
		t.Fatalf("List: %v, %d interfaces", err, len(info.Interfaces))
	}
}

