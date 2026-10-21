package diagnose

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestParseTCP(t *testing.T) {
	for _, s := range []string{"tcp:localhost:0", "tcp:localhost:65536", "tcp:bad", "http:localhost:80", "tcp::80"} {
		if _, err := ParseTCPCondition(s); err == nil {
			t.Errorf("accepted %q", s)
		}
	}
	for _, s := range []string{"tcp:localhost:5432", "tcp:[::1]:443"} {
		if _, err := ParseTCPCondition(s); err != nil {
			t.Errorf("rejected %q: %v", s, err)
		}
	}
}
func TestTCPMetAndUnmet(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	condition := "tcp:" + ln.Addr().String()
	r, err := CheckTCP(context.Background(), condition, time.Second)
	if err != nil || r.Status != "met" {
		t.Fatalf("%+v %v", r, err)
	}
	ln.Close()
	r, err = CheckTCP(context.Background(), condition, time.Second)
	if err != nil || r.Status != "unmet" {
		t.Fatalf("%+v %v", r, err)
	}
}
