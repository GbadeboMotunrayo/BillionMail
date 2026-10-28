package public

import "strings"

// IsImplicitTLSPort reports whether an SMTP port speaks TLS from the first byte.
// 465 is the standard; providers also publish variants such as 2465 (#379).
func IsImplicitTLSPort(port string) bool {
	port = strings.TrimSpace(port)
	return port == "465" || (len(port) > 3 && strings.HasSuffix(port, "465"))
}
