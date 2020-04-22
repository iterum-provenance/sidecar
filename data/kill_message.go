package data

import (
	"encoding/json"

	"github.com/iterum-provenance/sidecar/transmit"
)

// KillMessage is a message noting that either the sidecar or the program should stop
// a KillMessage indicates the final message on a socket after which the connection can be broken
type KillMessage struct {
	Status string `json:"status"`
}

// Serialize tries to transform `km` into a json encoded bytearray. Errors on failure
func (km *KillMessage) Serialize() (data []byte, err error) {
	data, err = json.Marshal(km)
	if err != nil {
		err = transmit.ErrSerialization(err)
	}
	return

}

// Deserialize tries to decode a json encoded byte array into `km`. Errors on failure
func (km *KillMessage) Deserialize(data []byte) (err error) {
	err = json.Unmarshal(data, km)
	if err != nil {
		err = transmit.ErrSerialization(err)
	}
	return
}