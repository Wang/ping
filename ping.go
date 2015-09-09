package ping

import (
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

var (
	mu         sync.Mutex
	queue      *Queue
	seq        int
	conn       *icmp.PacketConn
	pingServer *Server

	ProcessId int
	Debug     bool
)

type PingCmd struct {
	ip       string
	timeout  int
	sendTime time.Time
	done     chan string
	key      string
}

func init() {
	ProcessId = os.Getpid() & 0xffff
}

func SetDevice(d string) {
	var err error
	conn, err = icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		panic(err.Error())
	}
	queue = NewQueue()

	pingServer = NewServer(d, ProcessId)
	pingServer.Start()

	c := make(chan IcmpEchoReply)
	pingServer.ReceivingIcmp(c)
	go func(c <-chan IcmpEchoReply) {
		for e := range c {
			k := fmt.Sprintf("%s:%d", e.srcIp, e.seq)
			//log.Println(e.srcIp, ">", e.dstIp, e.seq, k)
			p, ok := queue.Get(k)
			if ok {
				p.done <- k
			}
		}
	}(c)
}

func Run(address string, timeout int) error {
	wm, pcmd, err := createIcmp(address, timeout)
	if err != nil {
		return err
	}
	defer queue.Del(pcmd.key)

	wb, err := wm.Marshal(nil)
	if err != nil {
		return err
	}
	if n, err := conn.WriteTo(wb, &net.IPAddr{IP: net.ParseIP(address)}); err != nil {
		return err
	} else if n != len(wb) {
		return errors.New("connection write to size error")
	}

	select {
	case <-time.After(time.Duration(timeout) * time.Millisecond):
		return errors.New("ping timeout")
	case <-pcmd.done:
		return nil
	}
}

func createIcmp(address string, timeout int) (icmp.Message, *PingCmd, error) {
	mu.Lock()
	defer mu.Unlock()
	if seq == 65535 {
		seq = 0
	}
	i := 0
	for ; i < 3; i++ {
		seq++
		key := fmt.Sprintf("%s:%d", address, seq)
		p := PingCmd{
			ip:       address,
			timeout:  timeout,
			done:     make(chan string, 1),
			sendTime: time.Now(),
			key:      key,
		}
		if ok := queue.Add(key, &p); ok {
			wm := icmp.Message{
				Type: ipv4.ICMPTypeEcho, Code: 0,
				Body: &icmp.Echo{
					ID:   ProcessId,
					Seq:  seq,
					Data: []byte("fast ping it!"),
				},
			}

			return wm, &p, nil
		}
	}

	return icmp.Message{}, &PingCmd{}, errors.New("Ping queue full")
}
