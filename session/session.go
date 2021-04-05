package session

import (
	"weavelab.xyz/ethr/ethr"
)


func CreateAckMsg () (msg *ethr.Msg){
	msg = &ethr.Msg{Version: 0, Type: ethr.Ack}
	msg.Ack = &ethr.MsgAck{}
	return
}