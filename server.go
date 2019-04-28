package main

import (
	"crypto/tls"

	"log"
	"net"
)

func serverconnet(remote string) net.Conn {
	conn, err := net.Dial("tcp", remote)
	if err != nil {
		log.Println(err.Error())
		return nil
	}
	log.Printf("server connect %s success!\n", remote)
	return conn
}

func serverreader(t *MessageTrans, channel *Channel) {
	var buff [MAX_BUF_SIZE]byte

	for {
		cnt, err := channel.Read(buff[:])
		if err != nil {
			log.Println(err.Error())
			channelclose(channel.chanid, t)
			return
		}

		rsp := &MessageRsponse{ChanID: channel.chanid, MsgType: CONNECT, Body: buff[:cnt]}
		err = t.MessageRsponseSend(rsp)
		if err != nil {
			log.Println(err.Error())
			return
		}
	}
}

func channelclose(chanid uint32, t *MessageTrans) {
	rsq := &MessageRsponse{ChanID: chanid, MsgType: CLOSE, Body: make([]byte, 0)}
	t.MessageRsponseSend(rsq)
}

func serverchannelpools(conn net.Conn) {
	channelpool := NewChannelPool()
	reader := NewMessageTrans(conn)
	writer := NewMessageTrans(conn)

	log.Println("server recv connect success!")

	for {
		req, err := reader.MessageRequestRecv()
		if err != nil {
			log.Println(err.Error())
			break
		}

		channel := channelpool.Find(req.ChanID)
		if channel == nil {
			chanconn := serverconnet(req.RemoteAdd)
			if chanconn != nil {
				channel = channelpool.Add(req.ChanID, req.RemoteAdd, chanconn)
			} else {
				channelclose(req.ChanID, writer)
			}
			go serverreader(writer, channel)
		}

		if req.MsgType == CLOSE {
			channel.Close()
		} else {
			channel.Write(req.Body)
		}
	}
	conn.Close()
	channelpool.Close()
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

		go serverchannelpools(conn)
	}

	return nil
}
