package main

import (
	"errors"
	"net"
	"sync"
	"time"
)

const (
	MAX_BUF_SIZE      = 4 * 1024        // 缓冲区大小(单位：byte)
	MAGIC_FLAG        = 0x98b7f30a      // 校验魔术字
	MSG_HEAD_LEN      = 2 * 4           // 消息头长度
	MAS_TRANS_TIMEOUT = 1 * time.Minute // 传输超时时间
)

type MessageType uint32

const (
	_ MessageType = iota
	CONNECT
	CLOSE
)

type MessageRequest struct {
	ChanID    uint32
	MsgType   MessageType
	RemoteAdd string
	Body      []byte
}

type MessageRsponse struct {
	ChanID  uint32
	MsgType MessageType
	Body    []byte
}

type MessageTrans struct {
	sync.Mutex
	exit    bool
	buffer  []byte
	index   int
	timeout time.Duration
	conn    net.Conn
}

func NewMessageTrans(conn net.Conn) *MessageTrans {
	return &MessageTrans{buffer: make([]byte, 64*1024), conn: conn, timeout: MAS_TRANS_TIMEOUT}
}

func TransferCoder(body []byte) []byte {
	trans := make([]byte, len(body)+MSG_HEAD_LEN)
	PutUint32(MAGIC_FLAG, trans[:4])
	PutUint32(uint32(len(body)), trans[4:8])
	copy(trans[MSG_HEAD_LEN:], body)
	return trans
}

func TransferDecoder(body []byte) []byte {
	return body[MSG_HEAD_LEN:]
}

func (t *MessageTrans) Close() {
	t.Lock()
	defer t.Unlock()

	t.conn.Close()
	t.exit = true
}

func (t *MessageTrans) MessageRequestSend(req *MessageRequest) error {
	t.Lock()
	defer t.Unlock()

	if t.exit {
		return errors.New("trans close!")
	}

	body, err := BinaryCoder(req)
	if err != nil {
		return errors.New("coder request failed!")
	}

	body = TransferCoder(body)
	var sendcnt int

	for {
		deadline := time.Now().Add(t.timeout)
		t.conn.SetWriteDeadline(deadline)

		cnt, err := t.conn.Write(body[sendcnt:])
		if err != nil {
			return err
		}

		sendcnt += cnt
		if sendcnt >= len(body) {
			break
		}
	}

	return nil
}

func (t *MessageTrans) MessageRequestRecv() (*MessageRequest, error) {
	t.Lock()
	defer t.Unlock()

	if t.exit {
		return nil, errors.New("trans close!")
	}

	var req MessageRequest

	for {
		deadline := time.Now().Add(t.timeout)
		t.conn.SetReadDeadline(deadline)

		cnt, err := t.conn.Read(t.buffer[t.index:])
		if err != nil {
			return nil, err
		}

		t.index += cnt

		if t.index < MSG_HEAD_LEN {
			continue
		}

		flag := GetUint32(t.buffer[0:4])
		Size := GetUint32(t.buffer[4:8])

		if flag != MAGIC_FLAG {
			return nil, errors.New("trans failed!")
		}

		if uint32(t.index) >= (Size + MSG_HEAD_LEN) {
			err := BinaryDecoder(t.buffer[MSG_HEAD_LEN:], &req)
			if err != nil {
				return nil, errors.New("decoder request failed!")
			}
			copy(t.buffer[0:], t.buffer[Size+MSG_HEAD_LEN:t.index])
			t.index -= int(Size + MSG_HEAD_LEN)
			break
		}
	}

	return &req, nil
}

func (t *MessageTrans) MessageRsponseSend(rsp *MessageRsponse) error {
	t.Lock()
	defer t.Unlock()

	if t.exit {
		return errors.New("trans close!")
	}

	body, err := BinaryCoder(rsp)
	if err != nil {
		return errors.New("coder rsponse failed!")
	}

	body = TransferCoder(body)
	var sendcnt int

	for {
		deadline := time.Now().Add(t.timeout)
		t.conn.SetWriteDeadline(deadline)

		cnt, err := t.conn.Write(body[sendcnt:])
		if err != nil {
			return err
		}

		sendcnt += cnt
		if sendcnt >= len(body) {
			break
		}
	}

	return nil
}

func (t *MessageTrans) MessageRsponseRecv() (*MessageRsponse, error) {
	t.Lock()
	defer t.Unlock()

	if t.exit {
		return nil, errors.New("trans close!")
	}

	var rsp MessageRsponse

	for {
		deadline := time.Now().Add(t.timeout)
		t.conn.SetReadDeadline(deadline)

		cnt, err := t.conn.Read(t.buffer[t.index:])
		if err != nil {
			return nil, err
		}

		t.index += cnt

		if t.index < MSG_HEAD_LEN {
			continue
		}

		flag := GetUint32(t.buffer[0:4])
		Size := GetUint32(t.buffer[4:8])

		if flag != MAGIC_FLAG {
			return nil, errors.New("trans failed!")
		}

		if uint32(t.index) >= (Size + MSG_HEAD_LEN) {
			err := BinaryDecoder(t.buffer[MSG_HEAD_LEN:], &rsp)
			if err != nil {
				return nil, errors.New("decoder rsponse failed!")
			}
			t.index -= int(Size + MSG_HEAD_LEN)
			break
		}
	}

	return &rsp, nil
}
