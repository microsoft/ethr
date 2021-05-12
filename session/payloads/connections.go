package payloads

import "fmt"

type ConnectionsPerSecondPayload struct {
	Connections uint64
}

func (p ConnectionsPerSecondPayload) String() string {
	return fmt.Sprintf("connections: %d", p.Connections)
}
