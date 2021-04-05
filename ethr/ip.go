package ethr

type IPVersion int

const (
	IPAny IPVersion = -1
	IPv4 IPVersion = 4
	IPv6 IPVersion = 6
)


var CurrentIPVersion IPVersion = IPAny
func SetCurrentIPVersion(v IPVersion) {
	CurrentIPVersion = v
}