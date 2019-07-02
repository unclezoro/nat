package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/libp2p/go-reuseport"
)

var (
	address   = make(map[string]*net.TCPAddr)
	addresMux = sync.Mutex{}

	seedAddr  string
	localAddr string
)

func init() {
	flag.StringVar(&seedAddr, "seed", "", "seedAddr")
	flag.StringVar(&localAddr, "local", "", "localAddr")

}

func main() {
	flag.Parse()

	lc := net.ListenConfig{
		Control: reuseport.Control,
	}
	ctx := context.Background()
	ll, err := lc.Listen(ctx, "tcp", localAddr)
	if err != nil {
		panic(err)
	}
	d := net.Dialer{
		LocalAddr: ll.Addr(),
		Control:   reuseport.Control,
	}
	if seedAddr != "" {
		go clientPeerRoutine(d, seedAddr)
	}

	for {
		rc, err := ll.Accept()
		if err != nil {
			panic(err)
		}
		fmt.Printf("accept remote addr is %v\n", rc.RemoteAddr())
		go func(c net.Conn) {
			ra := c.RemoteAddr()
			if tcpAddr, ok := ra.(*net.TCPAddr); ok {
				addresMux.Lock()
				if _, exist := address[tcpAddr.String()]; exist {
					addresMux.Unlock()
					return
				} else {
					address[tcpAddr.String()] = tcpAddr
					addresMux.Unlock()
				}

			}
			tick := time.Tick(time.Second)
			for range tick {
				stop := func() bool {
					addresMux.Lock()
					defer addresMux.Unlock()
					for _, a := range address {
						if a.String() != c.RemoteAddr().String() {
							fmt.Println("send addr", "addr", a.String(), "to", c.RemoteAddr().String())
							bz, err := json.Marshal(a)
							if err != nil {
								panic(err)
							}
							_, err = c.Write(bz)
							if err != nil {
								fmt.Println("write failed", err)
								return true
							}
						}
					}
					return false
				}()
				if stop {
					return
				}
			}
		}(rc)
	}
}

func clientPeerRoutine(d net.Dialer, addr string) {
	var conn net.Conn
	var err error
	for i:=0;i<10;i++{
		conn, err = d.Dial("tcp", addr)
		if err == nil {
			break
		}
	}
	if err !=nil{
		panic(err)
	}
	ra := conn.RemoteAddr()
	if tcpAddr, ok := ra.(*net.TCPAddr); ok {
		addresMux.Lock()
		address[tcpAddr.String()] = tcpAddr
		addresMux.Unlock()
	}
	for {
		bz := make([]byte, 1024)
		n, err := conn.Read(bz)
		if err != nil {
			fmt.Println("read failed", err)
			return
		}
		raw := bz[:n]
		tcpAddr := net.TCPAddr{}
		err = json.Unmarshal(raw, &tcpAddr)
		if err != nil {
			panic(err)
		}
		addresMux.Lock()
		if _, exist := address[tcpAddr.String()]; !exist {
			fmt.Println("receive new address", tcpAddr.String())
			address[tcpAddr.String()] = &tcpAddr
			go clientPeerRoutine(d, tcpAddr.String())
		}
		addresMux.Unlock()
	}
}
