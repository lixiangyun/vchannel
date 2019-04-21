package main

import (
	"crypto/tls"
	"errors"
	"log"
	"net"
	"sync"
)

const (
	MAX_BUF_SIZE = 64 * 1024  // 缓冲区大小(单位：byte)
	MAGIC_FLAG   = 0x98b7f30a // 校验魔术字
	MSG_HEAD_LEN = 3 * 4      // 消息头长度
)

type TransferHeader struct {
	Flag uint32 // 魔术字
	Size uint32 // 内容长度
	Body []byte // 传输的内容
}

type MessageType uint32

const (
	_ MessageType = iota
	CONNECT
	RUNING
	CLOSE
)

type MessageRequest struct {
	ChanID    string
	MsgType   MessageType
	LocalAdd  string
	RemoteAdd string
	Body      []byte
}

type MessageRsponse struct {
	ChanID    string
	MsgType   MessageType
	LocalAdd  string
	RemoteAdd string
	Body      []byte
}

type Channel struct {
	sync.Mutex
	sync.WaitGroup
	id        string
	localadd  string
	remoteadd string
	exit      bool
	conn      net.Conn
	read      chan []byte
	write     chan []byte
}

type ChannelPool struct {
	channels map[string]Channel
	sync.RWMutex
}

func (c *Channel) Read() ([]byte, error) {

	if c.exit {
		return nil, errors.New("channel close")
	}
	body, ok := <-c.read
	if !ok {
		return nil, errors.New("channel close")
	}
	return body, nil
}

func (c *Channel) Write(body []byte) error {

	bodycpy := make([]byte, len(body))
	copy(bodycpy, body)

	c.Lock()
	defer c.Unlock()
	if c.exit {
		return errors.New("channel close")
	}
	c.write <- bodycpy
	return nil
}

func (c *Channel) Close() {
	c.Lock()
	if c.exit {
		c.Unlock()
		return
	}
	c.conn.Close()
	close(c.write)
	close(c.read)
	c.exit = true
	c.Unlock()

	c.Wait()
}

func NewChannel(id, localadd, remoteadd string, conn net.Conn) *Channel {
	channel := &Channel{id: id, localadd: localadd, remoteadd: remoteadd, conn: conn}
	channel.read = make(chan []byte, 128)
	channel.write = make(chan []byte, 128)
	return channel
}

func channelread(c *Channel) {
	defer c.Done()

	var body [MAX_BUF_SIZE]byte
	for {
		cnt, err := c.conn.Read(body[:])
		if err != nil {
			log.Println(err.Error())
			c.exit = true
			return
		}

		bodycpy := make([]byte, cnt)
		copy(bodycpy, body[:cnt])

		c.Lock()
		if c.exit {
			c.Unlock()
			return
		}
		c.read <- bodycpy
		c.Unlock()
	}
}

func channelwrite(c *Channel) {
	defer c.Done()

	for {
		body, ok := <-c.write
		if !ok {
			return
		}

		var sendcnt int
		for {
			cnt, err := c.conn.Write(body[sendcnt:])
			if err != nil {
				log.Println(err.Error())
				c.exit = true
				return
			}
			sendcnt += cnt
			if sendcnt == len(body) {
				break
			}
		}
	}
}

func (c *Channel) Start() {
	c.Add(2)
	go channelread(c)
	go channelwrite(c)
}

func (c *ChannelPool) Write(chanid string, body []byte) error {
	c.RLock()
	defer c.RUnlock()
	channel, ok := c.channels[chanid]
	if !ok {
		log.Printf("channel id %s is not exit!\n", chanid)
		return errors.New("channel is not exit")
	}
	return channel.Write(body)
}

func (c *ChannelPool) Del(chanid string) {
	c.Lock()
	defer c.Unlock()
	delete(c.channels, chanid)
}

func (c *ChannelPool) Connect(chanid, local, remote string, conn *net.Conn) *Channel {
	c.Lock()
	defer c.Unlock()
}

func (c *ChannelPool) Stop() {
	c.Lock()
	defer c.Unlock()
	for _, v := range c.channels {
		v.Close()
	}
	c.channels = nil
}

func serverchannel(conn net.Conn) {

	var buf [MAX_BUF_SIZE]byte
	chanpool := new(ChannelPool)

	for {
		cnt, err := conn.Read(buf[:])
		if err != nil {
			log.Println(err.Error())
			return
		}

		cnt++
	}

	chanpool.Stop()
}

func ServerStart() error {

	var tlsconfig *tls.Config

	svccfg := ServerCfgGet()

	tlscfg := TlsCfgGet(svccfg.Tlsname)
	if tlscfg != nil {
		tlsconfig = TlsServerConfig(tlscfg)
		if tlsconfig == nil {
			log.Fatalf("server load tls config failed! %v\n", tlscfg)
		}
	}

	lis, err := net.Listen("tcp", svccfg.Address)
	if err != nil {
		log.Fatalln(err.Error())
	}

	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Println(err.Error())
			continue
		}

		if tlsconfig != nil {
			conn = tls.Server(conn, tlsconfig)
		}

		go serverchannel(conn)
	}

	return nil
}
