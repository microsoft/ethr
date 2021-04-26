package tools

import "net"

func (t *Tools) LookupHopName(addr string) string {
	if addr == "" {
		return ""
	}
	names, err := net.LookupAddr(addr)
	if err == nil && len(names) > 0 {
		name := names[0]
		sz := len(name)

		if sz > 0 && name[sz-1] == '.' {
			name = name[:sz-1]
		}
		return name
	}
	return ""
}
