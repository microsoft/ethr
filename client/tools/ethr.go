package tools

import (
	"fmt"
	"net"
	"os"

	"weavelab.xyz/ethr/ethr"
	"weavelab.xyz/ethr/session"
)

func CreateAckMsg() (msg *ethr.Msg) {
	msg = &ethr.Msg{Version: 0, Type: ethr.Ack}
	msg.Ack = &ethr.MsgAck{}
	return
}

func CreateSynMsg(testID session.TestID, clientParam ethr.ClientParams) (msg *ethr.Msg) {
	msg = &ethr.Msg{Version: 0, Type: ethr.Syn}
	msg.Syn = &ethr.MsgSyn{}
	msg.Syn.TestID = testID
	msg.Syn.ClientParam = clientParam
	return
}

func (t Tools) HandshakeWithServer(test *session.Test, conn net.Conn) error {
	msg := CreateSynMsg(test.ID, test.ClientParam)
	err := t.Session.Send(conn, msg)
	if err != nil {
		return fmt.Errorf("failed to send SYN message: %w", err)
	}
	resp := t.Session.Receive(conn)
	if resp.Type != ethr.Ack {
		return fmt.Errorf("failed to receive ACK message: %w", os.ErrInvalid)
	}
	return nil
}
