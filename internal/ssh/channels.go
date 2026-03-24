package ssh

import (
	"encoding/binary"
	"io"
	"net"
	"strconv"

	"golang.org/x/crypto/ssh"
)

func (s *Server) HandleChannels(chans <-chan ssh.NewChannel) {
	for newChannel := range chans {
		if newChannel.ChannelType() != "direct-tcpip" {
			newChannel.Reject(ssh.UnknownChannelType, "only port forwarding allowed")
			continue
		}

		extra := newChannel.ExtraData()
		if len(extra) < 4 {
			newChannel.Reject(ssh.Prohibited, "invalid data")
			continue
		}

		l := int(binary.BigEndian.Uint32(extra[:4]))
		if len(extra) < 4+l+4 {
			newChannel.Reject(ssh.Prohibited, "invalid data")
			continue
		}

		host := string(extra[4 : 4+l])
		port := binary.BigEndian.Uint32(extra[4+l : 4+l+4])

		ch, reqs, err := newChannel.Accept()
		if err != nil {
			continue
		}

		go ssh.DiscardRequests(reqs)

		go s.relayChannel(ch, host, port)
	}
}

func (s *Server) relayChannel(ch ssh.Channel, host string, port uint32) {
	defer ch.Close()
	conn, err := net.Dial("tcp", net.JoinHostPort(host, strconv.Itoa(int(port))))
	if err != nil {
		return
	}
	defer conn.Close()

	go func() {
		io.Copy(conn, ch)
		conn.Close()
	}()
	io.Copy(ch, conn)
}
