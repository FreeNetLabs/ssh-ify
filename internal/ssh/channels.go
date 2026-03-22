package ssh

import (
	"encoding/binary"
	"io"
	"net"
	"strconv"

	"golang.org/x/crypto/ssh"
)

func HandleSSHChannels(chans <-chan ssh.NewChannel) {
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

		targetHost := string(extra[4 : 4+l])
		targetPort := binary.BigEndian.Uint32(extra[4+l : 4+l+4])

		ch, reqs, err := newChannel.Accept()
		if err != nil {
			continue
		}

		go ssh.DiscardRequests(reqs)

		go func() {
			defer ch.Close()
			targetConn, err := net.Dial("tcp", net.JoinHostPort(targetHost, strconv.Itoa(int(targetPort))))
			if err != nil {
				return
			}
			defer targetConn.Close()

			go func() {
				io.Copy(targetConn, ch)
				targetConn.Close()
			}()
			io.Copy(ch, targetConn)
		}()
	}
}
