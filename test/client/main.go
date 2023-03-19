package main

import (
	"encoding/binary"
	"fmt"
	"github.com/rs/zerolog/log"
	"io"
	"net"
	"time"
)

func main() {
	for i := 0; i < 10; i++ {
		fmt.Println("start client: ", i)
		go start()
		time.Sleep(time.Millisecond * 5)
	}
	time.Sleep(time.Second * 20)
}

func start() {
	addr, err := net.ResolveTCPAddr("tcp4", "localhost:10001")
	if err != nil {
		log.Fatal().
			Err(err).
			Str("addr", addr.String()).
			Msg("addr resolve error")
		return
	}
	tcp, err := net.DialTCP("tcp4", nil, addr)
	if err != nil {
		log.Fatal().
			Err(err).
			Str("addr", addr.String()).
			Msg("addr resolve error")
		return
	}
	go func() {
		for {
			b := make([]byte, 4)
			_, err := io.ReadFull(tcp, b)
			if err != nil {
				log.Error().
					Err(err).
					Msg("read msg error")
			}
			msglen := binary.BigEndian.Uint32(b)
			b = make([]byte, msglen)
			_, _ = io.ReadFull(tcp, b)
			ts := binary.BigEndian.Uint64(b)
			milli := time.Now().UnixMilli()
			fmt.Println(milli - int64(ts))
		}
	}()
	for {
		milli := time.Now().UnixMilli()
		bytes := []byte{0, 0, 0, 8, 0, 0, 0, 0, 0, 0, 0, 0}
		binary.BigEndian.PutUint64(bytes[4:], uint64(milli))
		_, err := tcp.Write(bytes)
		if err != nil {
			panic(err)
		}
		time.Sleep(time.Second)
	}
}
