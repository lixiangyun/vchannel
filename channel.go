package main

import (
	"errors"
	"log"
	"net"
	"sync"
)

type Channel struct {
	chanid    uint32
	remoteadd string
	exit      bool
	conn      net.Conn
}

type ChannelPool struct {
	channels map[uint32]*Channel
	sync.RWMutex
}

var (
	CHANNEL_CLOSE = errors.New("channel close!")
)

func (c *Channel) Read(body []byte) (int, error) {
	if c.exit {
		return 0, CHANNEL_CLOSE
	}
	return c.conn.Read(body)
}

func (c *Channel) Write(body []byte) error {
	if c.exit {
		return CHANNEL_CLOSE
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
	log.Printf("close %d channel!", c.chanid)
}

func NewChannelPool() *ChannelPool {
	return &ChannelPool{channels: make(map[uint32]*Channel)}
}

func (c *ChannelPool) Find(chanid uint32) *Channel {
	c.RLock()
	defer c.RUnlock()
	channel, _ := c.channels[chanid]
	return channel
}

func (c *ChannelPool) Del(chanid uint32) {
	c.Lock()
	defer c.Unlock()
	channel, _ := c.channels[chanid]
	if channel != nil {
		channel.Close()
		delete(c.channels, chanid)
	}
}

func (c *ChannelPool) Add(chanid uint32, remote string, conn net.Conn) *Channel {
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
	log.Println("channel pool destroy!")
}
