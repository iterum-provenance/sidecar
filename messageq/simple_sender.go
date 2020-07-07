package messageq

import (
	"sync"

	"github.com/iterum-provenance/iterum-go/transmit"
	"github.com/iterum-provenance/iterum-go/util"
	"github.com/prometheus/common/log"
	"github.com/streadway/amqp"
)

// SimpleSender is the structure that listens to a channel and redirects messages to rabbitMQ
type SimpleSender struct {
	ToSend      chan transmit.Serializable
	TargetQueue string
	BrokerURL   string
	messages    int
	publisher   *QPublisher
}

// NewSimpleSender creates a new sender which receives messages from a channel and sends them on the message queue.
func NewSimpleSender(toSend chan transmit.Serializable, brokerURL, targetQueue string) (sender SimpleSender) {
	return SimpleSender{
		toSend,
		targetQueue,
		brokerURL,
		0,
		nil,
	}
}

func (sender *SimpleSender) spawnPublisher(conn *amqp.Connection) {
	ch, err := conn.Channel() // Eventually closed by the QPublisher
	util.Ensure(err, "SimpleSender opened channel")
	pub := NewQPublisher(make(chan transmit.Serializable, 10), ch, sender.TargetQueue)
	sender.publisher = &pub
}

// StartBlocking listens to the channel, and send remoteFragments to the message queue on the OUTPUT_QUEUE queue.
func (sender *SimpleSender) StartBlocking() {
	wg := &sync.WaitGroup{}

	log.Infof("SimpleSender connecting to %s.\n", sender.BrokerURL)
	conn, err := amqp.Dial(sender.BrokerURL)
	util.Ensure(err, "SimpleSender connected to RabbitMQ")
	defer sender.Stop(wg, conn)

	sender.spawnPublisher(conn)
	sender.publisher.Start(wg)

	for msg := range sender.ToSend {
		log.Debugf("Simple sender got %v\n", msg)
		// hand to publisher
		sender.publisher.ToPublish <- msg
		sender.messages++
	}
	log.Infof("SimpleSender finishing up, published %v messages\n", sender.messages)
}

// Start asychronously calls StartBlocking via Gorouting
func (sender *SimpleSender) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		sender.StartBlocking()
	}()
}

// Stop finishes up and notifies the user of its progress
func (sender *SimpleSender) Stop(wg *sync.WaitGroup, conn *amqp.Connection) {
	close(sender.publisher.ToPublish)
	wg.Wait()
	conn.Close()
}
