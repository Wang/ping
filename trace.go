package ping

import (
	"log"
	"sync"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

type PayloadIcmpIpLayer struct {
	ip      layers.IPv4
	icmp    layers.ICMPv4
	payload gopacket.Payload
}

type IcmpEchoReply struct {
	srcIp string
	dstIp string
	id    int
	seq   int
}

type Server struct {
	pid    int
	device string
	wg     sync.WaitGroup

	stopReceiveChan  chan bool
	receiveParseChan chan []byte
	receiveIcmpChan  chan PayloadIcmpIpLayer
}

func NewServer(device string, pid int) *Server {
	s := Server{
		pid:              pid,
		device:           device,
		stopReceiveChan:  make(chan bool),
		receiveParseChan: make(chan []byte),
		receiveIcmpChan:  make(chan PayloadIcmpIpLayer, 1024),
	}

	return &s
}

func (s *Server) Start() {
	log.Print("ping trace Start\n")
	snaplen := 65536 // XXX make this a user specified option!
	filter := "icmp"

	handle, err := pcap.OpenLive(s.device, int32(snaplen), true, pcap.BlockForever)
	if err != nil {
		log.Fatal("error opening pcap handle: ", err)
	}
	if err := handle.SetBPFFilter(filter); err != nil {
		log.Fatal("error setting BPF filter: ", err)
	}

	s.wg.Add(2)
	s.parsingReplies()

	go func() {
		s.wg.Done()
	Loop:
		for {
			select {
			case <-s.stopReceiveChan:
				break Loop
			default:
				data, _, err := handle.ReadPacketData()
				if err != nil {
					continue
				}
				s.receiveParseChan <- data
			}
		}
		close(s.receiveParseChan)
		close(s.stopReceiveChan)
	}()

	s.wg.Wait()
}

func (s *Server) Stop() {
	log.Print("ping trace Stop\n")
	s.stopReceiveChan <- true
}

func (s *Server) parsingReplies() {
	var eth layers.Ethernet
	var ip layers.IPv4
	var icmp layers.ICMPv4
	var payload gopacket.Payload

	icmpParser := gopacket.NewDecodingLayerParser(layers.LayerTypeEthernet, &eth, &ip, &icmp, &payload)
	decoded := make([]gopacket.LayerType, 0, 4)

	go func() {
		s.wg.Done()
		defer close(s.receiveIcmpChan)
		for packet := range s.receiveParseChan {
			err := icmpParser.DecodeLayers(packet, &decoded)
			if err == nil {
				s.receiveIcmpChan <- PayloadIcmpIpLayer{
					ip:      ip,
					icmp:    icmp,
					payload: payload,
				}
			}
		}
	}()
}

func (s *Server) ReceivingIcmp(c chan<- IcmpEchoReply) {
	go func() {
		for bundle := range s.receiveIcmpChan {
			//log.Println("icmp packet received:", bundle.ip.SrcIP.String(), ">", bundle.ip.DstIP.String(), bundle.icmp.Seq, bundle.icmp.TypeCode.String())
			if bundle.icmp.TypeCode == 0 && s.pid == int(bundle.icmp.Id) { //ICMPv4TypeEchoReply
				c <- IcmpEchoReply{
					srcIp: bundle.ip.SrcIP.String(),
					dstIp: bundle.ip.DstIP.String(),
					id:    int(bundle.icmp.Id),
					seq:   int(bundle.icmp.Seq),
				}
			}
		}
	}()
}
