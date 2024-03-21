package p2p

import pubsub "github.com/libp2p/go-libp2p-pubsub"

type HandleSubscriptionMessage func([]byte, *pubsub.Topic)

type TopicEventMonitor interface {
	Subscribe() <-chan *TopicEvent
}

type TopicEvent struct {
	Topic string
	Type  TopicEventType
}

type TopicEventType string

const (
	TopicEventType_Upserted TopicEventType = "upserted"
	TopicEventType_Paused   TopicEventType = "paused"
)

type P2PDataHandler interface {
	Handle(input []byte) (outputs [][]byte)
}
