package socket

import (
	"net"

	desc "github.com/iterum-provenance/iterum-go/descriptors"
	"github.com/iterum-provenance/sidecar/garbage"

	"github.com/iterum-provenance/iterum-go/transmit"
	"github.com/prometheus/common/log"
)

// SendFileHandler is a handler function for a socket that sends files to the transformation step
func SendFileHandler(fragCollector garbage.FragmentCollector) func(socket Socket, conn net.Conn) {
	return func(socket Socket, conn net.Conn) {
		defer conn.Close()
		for {
			// Wait for the next job to come off the queue.
			msg, ok := <-socket.Channel

			if !ok { // channel was closed
				killMsg := desc.NewKillMessage()
				msg = &killMsg
			} else {
				// Meaning it definitely is a *desc.LocalFragDesc
				fragCollector.Track <- msg.(*desc.LocalFragmentDesc)
			}
			// Send the msg over the connection
			err := transmit.EncodeSend(conn, msg)
			// Error handling
			switch err.(type) {
			case *transmit.SerializationError:
				log.Warnf("Could not encode message due to '%v', skipping message", err)
				continue
			case *transmit.ConnectionError:
				log.Warnf("Closing connection towards due to '%v'", err)
				return
			default:
				log.Errorf("%v, closing connection", err)
				return
			case nil:
			}

			if !ok {
				socket.Stop()
				break
			}
		}
	}
}
