package main

import (
	"errors"
	"net"
	"sync"
)

type Channel struct {
	chanid    string
	remoteadd string
	exit      bool
	conn      net.Conn
}

type ChannelPool struct {
	channels map[string]*Channel
	sync.RWMutex
}

func (c *Channel) Read(body []byte) (int, error) {
	if c.exit {
		return 0, errors.New("channel close")
	}
	return c.conn.Read(body)
}

func (c *Channel) Write(body []byte) error {
	if c.exit {
		return errors.New("channel close")
	}
	var sendcnt int
	for {
		cnt, err := c.conn.Write(body[sendcnt:])
		if err != nil {
			return err
		}
		sendcnt += cnt
		if sendcnt == len(body) {
			break
		}
	}
	return nil
}

func (c *Channel) Close() {
	if c.exit {
		return
	}
	c.conn.Close()
	c.exit = true
}

func NewChannel(chanid, localadd, remoteadd string, conn net.Conn) *Channel {
	channel := &Channel{chanid: chanid, remoteadd: remoteadd, conn: conn}
	return channel
}

func NewChannelPool() *ChannelPool {
	return &ChannelPool{channels: make(map[string]*Channel)}
}

func (c *ChannelPool) Find(chanid string) *Channel {
	c.RLock()
	defer c.RUnlock()
	channel, _ := c.channels[chanid]
	return channel
}

func (c *ChannelPool) Del(chanid string) {
	c.Lock()
	defer c.Unlock()
	delete(c.channels, chanid)
}

func (c *ChannelPool) Add(chanid, remote string, conn net.Conn) *Channel {
	c.Lock()
	defer c.Unlock()

	channel := &Channel{chanid: chanid, remoteadd: remote, conn: conn}
	c.channels[chanid] = channel
	return channel
}

func (c *ChannelPool) Close() {
	c.Lock()
	defer c.Unlock()
	for _, v := range c.channels {
		v.Close()
	}
	c.channels = nil
}
