package main

import (
	"flag"
	"fmt"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"log"
	"net"
	"os"
	"time"
)

var (
	m           = flag.String("m", "GET", "")
)


const (
	ProtocolICMP = 1
	ListenAddr = "0.0.0.0"
	Timeout = 10 * time.Second
)



//TODO: Handle nil
//TODO: Handle panics


func Ping(addr string) (*net.IPAddr, time.Duration, error) {
	c, err := icmp.ListenPacket("ip4:icmp", ListenAddr)
	if err != nil {
		return nil, 0, err
	}
	defer c.Close()

	// Resolve any DNS (if used) and get the real IP of the target
	dst := resolveIPAddress(addr)

	// Make a new ICMP message
	message := icmp.Message{
		Type: ipv4.ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{
			ID: os.Getpid() & 0xffff, Seq: 1,
			Data: []byte(""),
		},
	}
	bytes, err := message.Marshal(nil)
	if err != nil {
		return dst, 0, err
	}

	start := time.Now()
	n, err := c.WriteTo(bytes, dst)
	if err != nil {
		return dst, 0, err
	} else if n != len(bytes) {
		return dst, 0, fmt.Errorf("got %v; want %v", n, len(bytes))
	}

	peer, duration, readMessage, ipAddr, t, err2 := readMessage( c, dst, n, start)
	if err2 != nil {
		return ipAddr, t, err2
	}
	switch readMessage.Type {
	case ipv4.ICMPTypeEchoReply:
		return dst, duration, nil
	default:
		return dst, 0, fmt.Errorf("got %+v from %v; want echo reply", readMessage, peer)
	}
}

func readMessage(c *icmp.PacketConn, dst *net.IPAddr, n int, start time.Time) (net.Addr, time.Duration, *icmp.Message, *net.IPAddr, time.Duration, error) {
	reply := make([]byte, 1500)
	err := c.SetReadDeadline(time.Now().Add(Timeout))
	if err != nil {
		return nil, 0, nil, dst, 0, err
	}
	n, peer, err := c.ReadFrom(reply)
	if err != nil {
		return nil, 0, nil, dst, 0, err
	}
	duration := time.Since(start)

	rm, err := icmp.ParseMessage(ProtocolICMP, reply[:n])
	if err != nil {
		return nil, 0, nil, dst, 0, err
	}
	return peer, duration, rm, nil, 0, nil
}

func resolveIPAddress(addr string) *net.IPAddr {
	dst, _ := net.ResolveIPAddr("ip4", addr)
	return dst
}


func main() {
	p := func(addr string){
		dst, dur, err := Ping(addr)
		if err != nil {
			log.Printf("Ping %s (%s): %s\n", addr, dst, err)
			return
		}
		log.Printf("Ping %s (%s): %s\n", addr, dst, dur)
	}
	flag.Parse()
	if hostname := flag.Arg(0); hostname != "" {
		fmt.Println(hostname)
		p(hostname)
	} else {
		//TODO: Handle helper
		fmt.Println("ERROR")
	}

	
}
