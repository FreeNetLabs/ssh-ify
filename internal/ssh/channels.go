package ssh

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"sync"

	"golang.org/x/crypto/ssh"
)

func HandleSSHChannels(chans <-chan ssh.NewChannel) {
	for newChannel := range chans {
		if !isDirectTCPIPChannel(newChannel) {
			log.Printf("HandleChannels: Unknown channel type: %s", newChannel.ChannelType())
			newChannel.Reject(ssh.UnknownChannelType, "only port forwarding allowed")
			continue
		}

		targetHost, targetPort, err := parseDirectTCPIPExtra(newChannel.ExtraData())
		if err != nil {
			log.Printf("HandleChannels: %v", err)
			newChannel.Reject(ssh.Prohibited, err.Error())
			continue
		}

		ch, reqs, err := newChannel.Accept()
		if err != nil {
			log.Printf("HandleChannels: Error accepting channel: %v", err)
			continue
		}
		go ssh.DiscardRequests(reqs)
		go handlePortForwarding(targetHost, targetPort, ch)
	}
}

func forwardData(ch ssh.Channel, targetConn net.Conn, addr string) {
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, err := io.Copy(targetConn, ch)
		if err != nil && err != io.EOF {
			log.Printf("forwardChannel: Error copying SSH->%s: %v", addr, err)
		}
	}()
	go func() {
		defer wg.Done()
		_, err := io.Copy(ch, targetConn)
		if err != nil && err != io.EOF {
			log.Printf("forwardChannel: Error copying %s->SSH: %v", addr, err)
		}
	}()
	wg.Wait()
	targetConn.Close()
	ch.Close()
}

func isDirectTCPIPChannel(newChannel ssh.NewChannel) bool {
	return newChannel.ChannelType() == "direct-tcpip"
}

func parseDirectTCPIPExtra(extra []byte) (string, uint32, error) {
	if len(extra) < 4 {
		return "", 0, fmt.Errorf("invalid direct-tcpip request: insufficient data for host length")
	}
	l := int(binary.BigEndian.Uint32(extra[:4]))
	if len(extra) < 4+l+4 {
		return "", 0, fmt.Errorf("invalid direct-tcpip request: insufficient data for host and port")
	}
	targetHost := string(extra[4 : 4+l])
	portOffset := 4 + l
	targetPort := binary.BigEndian.Uint32(extra[portOffset : portOffset+4])
	return targetHost, targetPort, nil
}

func handlePortForwarding(targetHost string, targetPort uint32, ch ssh.Channel) {
	defer ch.Close()
	addr := net.JoinHostPort(targetHost, strconv.Itoa(int(targetPort)))
	targetConn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Printf("HandleChannels: Error connecting to target %s: %v", addr, err)
		return
	}
	forwardData(ch, targetConn, addr)
}
