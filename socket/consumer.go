package socket

import (
	"fmt"
	"net"

	desc "github.com/iterum-provenance/iterum-go/descriptors"
	"github.com/iterum-provenance/iterum-go/transmit"

	"github.com/prometheus/common/log"
)

// ProcessedFileHandler is a handler for a socket that receives processed files from the transformation step
func ProcessedFileHandler(acknowledger chan transmit.Serializable) func(socket Socket, conn net.Conn) {
	return func(socket Socket, conn net.Conn) {
		defer conn.Close()
		for {
			encMsg, err := transmit.ReadMessage(conn)
			if err != nil {
				log.Warnf("Failed to read, closing connection towards due to '%v'", err)
				return
			}

			fragMsg := fragmentDesc{}
			errFrag := fragMsg.Deserialize(encMsg)
			doneMsg := desc.FinishedFragmentMessage{}
			errDone := doneMsg.Deserialize(encMsg)
			killMsg := desc.KillMessage{}
			errKill := killMsg.Deserialize(encMsg)

			if errFrag == nil {
				// Default behaviour
				// unwrap socket fragmentDesc into general type before posting on output
				lfd := fragMsg.LocalFragmentDesc
				socket.Channel <- &lfd
			} else if errDone == nil {
				acknowledger <- &doneMsg
			} else if errKill == nil {
				log.Info("Received kill message, stopping consumer...")
				defer socket.Stop()
				defer close(socket.Channel)
				defer close(acknowledger)
				return
			} else {
				// Error handling
				switch errFrag.(type) {
				case *transmit.SerializationError:
					log.Fatalf("Could not decode message due to '%v'", errFrag)
					continue
				default:
					log.Errorf("'%v', closing connection", errFrag)
					return
				}
			}

		}
	}
}

// Consumer is a dummy setup to help test socket
func Consumer(channel chan transmit.Serializable) {
	for {
		msg, ok := <-channel
		if !ok {
			return
		}
		fragDesc := msg.(*fragmentDesc)
		fmt.Printf("Received: '%v'\n", *fragDesc)
	}
}
