package ethr

type MsgType uint32

const (
	Inv MsgType = iota
	Syn
	Ack
)

type MsgVer uint32

type Msg struct {
	Version MsgVer
	Type    MsgType
	Syn     *MsgSyn
	Ack     *MsgAck
}

type MsgSyn struct {
	TestID      TestID
	ClientParam ClientParams
}

type MsgAck struct {
}
