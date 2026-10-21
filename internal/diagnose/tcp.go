package diagnose

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// TCPResult describes only whether a TCP handshake succeeded, not application health.
type TCPResult struct {
	Condition string     `json:"condition"`
	Address   string     `json:"address"`
	Status    string     `json:"status"` // met, unmet, unknown
	Evidence  []Evidence `json:"evidence"`
	Unknowns  []string   `json:"unknowns"`
	NextSteps []string   `json:"next_steps"`
}

func ParseTCPCondition(s string) (string, error) {
	if !strings.HasPrefix(s, "tcp:") {
		return "", fmt.Errorf("expected tcp:host:port")
	}
	raw := strings.TrimPrefix(s, "tcp:")
	host, port, err := net.SplitHostPort(raw)
	if err != nil || host == "" {
		return "", fmt.Errorf("invalid TCP condition: use tcp:host:port (IPv6: tcp:[::1]:port)")
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return "", fmt.Errorf("port must be between 1 and 65535")
	}
	return net.JoinHostPort(host, strconv.Itoa(n)), nil
}

func CheckTCP(ctx context.Context, condition string, timeout time.Duration) (TCPResult, error) {
	addr, err := ParseTCPCondition(condition)
	if err != nil {
		return TCPResult{}, err
	}
	if timeout <= 0 || timeout > 30*time.Second {
		return TCPResult{}, fmt.Errorf("timeout must be >0 and <=30s")
	}
	r := TCPResult{Condition: condition, Address: addr, Status: "unknown", Evidence: []Evidence{}, Unknowns: []string{}, NextSteps: []string{}}
	c, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	conn, err := (&net.Dialer{}).DialContext(c, "tcp", addr)
	if err == nil {
		_ = conn.Close()
		r.Status = "met"
		r.Evidence = append(r.Evidence, Evidence{ID: "E001", Kind: "observed", Source: "TCP connect", Claim: "TCP handshake succeeded to " + addr})
		r.Unknowns = append(r.Unknowns, "A TCP handshake does not prove application protocol readiness or service health.")
		return r, nil
	}
	var op *net.OpError
	if errors.As(err, &op) && errors.Is(err, syscall.ECONNREFUSED) {
		r.Status = "unmet"
		r.Evidence = append(r.Evidence, Evidence{ID: "E001", Kind: "observed", Source: "TCP connect", Claim: "Connection refused by network stack for " + addr})
		r.Unknowns = append(r.Unknowns, "Cannot determine whether no listener exists, a firewall rejected the connection, or another component refused it.")
		r.NextSteps = append(r.NextSteps, "Inspect listeners and firewall rules on the target host.")
	} else {
		r.Unknowns = append(r.Unknowns, "TCP connect failed without a conclusive refusal: "+err.Error())
		r.NextSteps = append(r.NextSteps, "Check name resolution, routing, filtering, and reachability.")
	}
	return r, nil
}
