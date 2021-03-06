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
	TTL= flag.Int("m", 64, "ttl")
)


var usage = `Usage: PingClone [options...] <hostname/ip>
Options:
  -m  TTL for packet. Default is 64.
`

const (
	ProtocolICMPV4 = 1
	ProtocolICMPV6 = 58
	ListenAddr     = "0.0.0.0"
	Timeout        = 1 * time.Second
	Interval = 1*time.Second
)


//TODO: Handle nil
//TODO: Handle panics
//TODO: In the real world, I would check the sequence number and dest to verify it

func Ping(addr string) (*net.IPAddr, time.Duration, error, bool) {
	c, err := icmp.ListenPacket("ip4:icmp", ListenAddr)

	if err != nil {
		return nil, 0, err, false
	}
	defer c.Close()

	dst := resolveIPAddress(addr)

	message := icmp.Message{
		Type: ipv4.ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{
			ID: os.Getpid() & 0xffff, Seq: 1,
			Data: []byte(""),
		},
	}
	start := time.Now()
	bytes, err := message.Marshal(nil)
	if err != nil {
		return dst, 0, err, false
	}

	pc := c.IPv4PacketConn()
	pc.SetTTL(*TTL)
	n, err := pc.WriteTo(bytes, nil, dst)
	if err != nil {
		return dst, 0, err, false
	}

	peer, duration, readMessage, ipAddr, t, err, loss := readMessage( c, dst, n, start)
	if err != nil {
		return ipAddr, t, err, loss
	}
	switch readMessage.Type {
	case ipv4.ICMPTypeEchoReply:
		return dst, duration, nil, loss
	case ipv4.ICMPTypeTimeExceeded:
		return dst, duration, fmt.Errorf("Time exceeded"), loss
	default:
		return dst, 0, fmt.Errorf("got %+v from %v; want echo reply", readMessage, peer), loss
	}
}

func readMessage(c *icmp.PacketConn, dst *net.IPAddr, n int, start time.Time) (net.Addr, time.Duration, *icmp.Message, *net.IPAddr, time.Duration, error, bool) {
	reply := make([]byte, 1500)
	err := c.SetReadDeadline(time.Now().Add(Timeout))
	if err != nil {
		return nil, 0, nil, dst, 0, err, false
	}
	n, peer, err := c.ReadFrom(reply)
	if err != nil {
		return nil, 0, nil, dst, 0, err, true
	}
	duration := time.Since(start)

	rm, err := icmp.ParseMessage(ProtocolICMPV4, reply[:n])
	if err != nil {
		return nil, 0, nil, dst, 0, err, false
	}
	return peer, duration, rm, nil, 0, nil, false
}

func resolveIPAddress(addr string) *net.IPAddr {
	dst, _ := net.ResolveIPAddr("ip4", addr)
	return dst
}

func wrappedPing(addr string) bool {
	dst, dur, err, loss := Ping(addr)
	if err != nil {
		log.Printf("Ping %s (%s): %s\n", addr, dst, err)
	} else {
		log.Printf("Ping %s (%s): %s\n", addr, dst, dur)
	}
	return loss
}

func main() {
	flag.Parse()

	if hostname := flag.Arg(0); hostname != "" {
		packets_transmitted := 0
		packets_loss := 0
		for {
			packets_transmitted += 1
			loss := wrappedPing(hostname)
			if loss {
				packets_loss += 1
			}
			log.Printf("%d packets transmitted, %d packets received, %.2f%% packet loss", packets_transmitted, packets_transmitted - packets_loss, (100.0 * float64(packets_loss)/float64(packets_transmitted)))
			time.Sleep(Interval)
		}
	} else {
		//TODO: Handle helper
		usageAndExit()
	}
}
func usageAndExit() {
	flag.Usage()
	fmt.Fprintf(os.Stderr, "\n")
	os.Exit(1)
}

