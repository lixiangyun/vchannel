package main

import (
	"crypto/tls"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

func clientconnect() net.Conn {

	var tlsconfig *tls.Config

	clicfg := ClientCfgGet()

	tlscfg := TlsCfgGet(clicfg.Tlsname)
	if tlscfg != nil {
		tlsconfig = TlsClientConfig(tlscfg)
		if tlsconfig == nil {
			log.Fatalf("tls %v client config load failed!\n", tlscfg)
		}
	}

	conn, err := net.Dial("tcp", clicfg.Address)
	if err != nil {
		log.Println(err.Error())
		return nil
	}

	if tlsconfig != nil {
		conn = tls.Client(conn, tlsconfig)
	}

	log.Println("client connect success.")

	return conn
}

type clientchannel struct {
	sync.Mutex
	chanid      uint32
	reader      *MessageTrans
	writer      *MessageTrans
	channelpool *ChannelPool
}

var globalclient clientchannel

func ClientWrite(req *MessageRequest) {
	writer := globalclient.writer
	if writer == nil {
		log.Println("client writer close.")
		return
	}
	err := writer.MessageRequestSend(req)
	if err != nil {
		go clientclose()
	}
}

func clientclose() {
	globalclient.Lock()
	defer globalclient.Unlock()

	if globalclient.reader != nil {
		globalclient.reader.Close()
		globalclient.reader = nil
	}

	if globalclient.writer != nil {
		globalclient.writer.Close()
		globalclient.writer = nil
	}

	if globalclient.channelpool != nil {
		globalclient.channelpool.Close()
		globalclient.channelpool = nil
	}

	log.Println("client connect success.")
}

func clientsendclose(chanid uint32) {
	req := &MessageRequest{ChanID: chanid, MsgType: CLOSE, Body: make([]byte, 0)}
	ClientWrite(req)
}

func clientreader(reader *MessageTrans) {
	for {

		rsp, err := reader.MessageRsponseRecv()
		if err != nil {
			go clientclose()
			return
		}

		channel := clientchannelget(rsp.ChanID)
		if channel == nil {
			continue
		}

		if rsp.MsgType == CLOSE {
			channel.Close()
		} else {
			channel.Write(rsp.Body)
		}
	}
}

func clientchannelget(chanid uint32) *Channel {
	globalclient.Lock()
	defer globalclient.Unlock()

	if globalclient.channelpool == nil {
		return nil
	}

	return globalclient.channelpool.Find(chanid)
}

func clientchanneldel(chanid uint32) {
	globalclient.Lock()
	defer globalclient.Unlock()

	if globalclient.channelpool != nil {
		globalclient.channelpool.Del(chanid)
	}
}

func clientchannelnew(chanid uint32, remoteadd string, conn net.Conn) *Channel {

	globalclient.Lock()
	defer globalclient.Unlock()

	if globalclient.reader == nil || globalclient.writer == nil {
		conn := clientconnect()
		if conn == nil {
			return nil
		}
		globalclient.reader = NewMessageTrans(conn)
		globalclient.writer = NewMessageTrans(conn)

		go clientreader(globalclient.reader)
	}

	if globalclient.channelpool == nil {
		globalclient.channelpool = NewChannelPool()
	}

	return globalclient.channelpool.Add(chanid, remoteadd, conn)
}

func ChannelProcess(remoteadd string, conn net.Conn) {

	chanid := atomic.AddUint32(&globalclient.chanid, 1)
	channel := clientchannelnew(chanid, remoteadd, conn)

	if channel == nil {
		conn.Close()
		log.Println("channel new failed!")
		return
	}

	log.Printf("channel %d new success!", channel.chanid)

	var buff [MAX_BUF_SIZE]byte
	for {
		cnt, err := channel.Read(buff[:])
		if err != nil {
			log.Println(err.Error())
			break
		} else {
			req := &MessageRequest{ChanID: chanid,
				MsgType: CONNECT, RemoteAdd: remoteadd,
				Body: buff[:cnt]}
			ClientWrite(req)
		}
	}

	clientsendclose(chanid)
	clientchanneldel(chanid)
}

func ChannelStart(localadd string, remoteadd string) {
	lis, err := net.Listen("tcp", localadd)
	if err != nil {
		log.Fatalln(err.Error())
	}

	log.Printf("channel %s listen.\n", localadd)

	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Println(err.Error())
			continue
		}

		log.Printf("channel %s->%s start.\n", localadd, remoteadd)
		go ChannelProcess(remoteadd, conn)
	}
}

func ClientStart() {

	chancfg := ChannelCfgGet()
	if len(chancfg) == 0 {
		log.Println("channel have not config.")
		return
	}

	for _, v := range chancfg {
		go ChannelStart(v.Local, v.Remote)
	}

	for {
		time.Sleep(60 * time.Second)
	}

}
