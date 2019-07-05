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

	seedAddr1 string
	seedPort1 int
	seedAddr2 string
	seedPort2 int

	localAddr string
)

func init() {
	flag.StringVar(&seedAddr1, "seed1", "", "seedAddr1")
	flag.IntVar(&seedPort1, "seed_port1", 0, "seed_port1")
	flag.StringVar(&seedAddr2, "seed2", "", "seedAddr2")
	flag.IntVar(&seedPort2, "seed_port2", 0, "seed_port2")
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
	if seedAddr1 != "" {
		tcpSeed1, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", seedAddr1, seedPort1))
		if err != nil {
			panic(err)
		}
		go clientPeerRoutine(d, *tcpSeed1)
	}
	if seedAddr2 != "" {
		tcpSeed2, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", seedAddr2, seedPort2))
		if err != nil {
			panic(err)
		}
		go clientPeerRoutine(d, *tcpSeed2)
	}

	go func() {
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
									delete(address, c.RemoteAddr().String())
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
	}()

	select {}

}

func clientPeerRoutine(d net.Dialer, addr net.TCPAddr) {
	var conn net.Conn
	var err error
	fmt.Println("dial start", addr.String())

	for i := 0; i < 3; i++ {
		conn, err = d.Dial("tcp", addr.String())
		if err == nil {
			break
		}
	}
	if err != nil {
		panic(err)
	}
	fmt.Println("dial success", addr.String())
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
			fmt.Println(string(raw))
			panic(err)
		}
		addresMux.Lock()
		if _, exist := address[tcpAddr.String()]; !exist {
			fmt.Println("receive new address", tcpAddr.String())
			address[tcpAddr.String()] = &tcpAddr
			go clientPeerRoutine(d, tcpAddr)
		}
		addresMux.Unlock()
	}
}
