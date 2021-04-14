package session

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"net"
	"os"

	"weavelab.xyz/ethr/ethr"
)

func CreateAckMsg() (msg *ethr.Msg) {
	msg = &ethr.Msg{Version: 0, Type: ethr.Ack}
	msg.Ack = &ethr.MsgAck{}
	return
}

func CreateSynMsg(testID TestID, clientParam ethr.ClientParams) (msg *ethr.Msg) {
	msg = &ethr.Msg{Version: 0, Type: ethr.Syn}
	msg.Syn = &ethr.MsgSyn{}
	msg.Syn.TestID = testID
	msg.Syn.ClientParam = clientParam
	return
}

func (s Session) HandshakeWithServer(test *Test, conn net.Conn) error {
	msg := CreateSynMsg(test.ID, test.ClientParam)
	err := s.Send(conn, msg)
	if err != nil {
		return fmt.Errorf("failed to send SYN message: %w", err)
	}
	resp := s.Receive(conn)
	if resp.Type != ethr.Ack {
		return fmt.Errorf("failed to receive ACK message: %w", os.ErrInvalid)
	}
	return nil
}

func (s Session) HandshakeWithClient(conn net.Conn) (testID TestID, clientParam ethr.ClientParams, err error) {
	msg := s.Receive(conn)
	if msg.Type != ethr.Syn {
		err = os.ErrInvalid
		return
	}
	testID = msg.Syn.TestID
	clientParam = msg.Syn.ClientParam
	ack := CreateAckMsg()
	err = s.Send(conn, ack)
	return
}

func (s Session) Receive(conn net.Conn) (msg *ethr.Msg) {
	msg = &ethr.Msg{}
	msg.Type = ethr.Inv
	msgBytes := make([]byte, 4)
	_, err := io.ReadFull(conn, msgBytes)
	if err != nil {
		Logger.Debug("Error receiving message on control channel. Error: %v", err)
		return
	}
	msgSize := binary.BigEndian.Uint32(msgBytes[0:])
	// TODO: Assuming max ethr message size as 16K sent over gob.
	if msgSize > 16384 {
		return
	}
	msgBytes = make([]byte, msgSize)
	_, err = io.ReadFull(conn, msgBytes)
	if err != nil {
		Logger.Debug("Error receiving message on control channel. Error: %v", err)
		return
	}
	msg = decodeMsg(msgBytes)
	return
}

func (s Session) ReceiveFromBuffer(msgBytes []byte) (msg *ethr.Msg) {
	msg = decodeMsg(msgBytes)
	return
}

func (s Session) Send(conn net.Conn, msg *ethr.Msg) (err error) {
	msgBytes, err := encodeMsg(msg)
	if err != nil {
		Logger.Debug("Error sending message on control channel. Message: %v, Error: %v", msg, err)
		return
	}
	msgSize := len(msgBytes)
	tempBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(tempBuf[0:], uint32(msgSize))
	_, err = conn.Write(tempBuf)
	if err != nil {
		Logger.Debug("Error sending message on control channel. Message: %v, Error: %v", msg, err)
	}
	_, err = conn.Write(msgBytes)
	if err != nil {
		Logger.Debug("Error sending message on control channel. Message: %v, Error: %v", msg, err)
	}
	return err
}

func decodeMsg(msgBytes []byte) (msg *ethr.Msg) {
	msg = &ethr.Msg{}
	buffer := bytes.NewBuffer(msgBytes)
	decoder := gob.NewDecoder(buffer)
	err := decoder.Decode(msg)
	if err != nil {
		Logger.Debug("Failed to decode message using Gob: %v", err)
		msg.Type = ethr.Inv
	}
	return
}

func encodeMsg(msg *ethr.Msg) (msgBytes []byte, err error) {
	var writeBuffer bytes.Buffer
	encoder := gob.NewEncoder(&writeBuffer)
	err = encoder.Encode(msg)
	if err != nil {
		Logger.Debug("Failed to encode message using Gob: %v", err)
		return
	}
	msgBytes = writeBuffer.Bytes()
	return
}
