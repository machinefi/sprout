package p2p

import (
	"context"
	"log/slog"
	"sync"

	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/muxer/yamux"
	"github.com/pkg/errors"
)

func NewPubSubs(topicEventMonitor TopicEventMonitor, handler P2PDataHandler, bootNodeMultiaddr string, iotexChainID int) (*PubSubs, error) {
	ctx := context.Background()
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"), libp2p.Muxer("/yamux/2.0.0", yamux.DefaultTransport))
	if err != nil {
		return nil, errors.Wrap(err, "new libp2p host failed")
	}

	sub, err := pubsub.NewGossipSub(ctx, h, pubsub.WithMaxMessageSize(2*pubsub.DefaultMaxMessageSize))
	if err != nil {
		return nil, errors.Wrap(err, "new gossip subscription failed")
	}
	if err := discoverPeers(ctx, h, bootNodeMultiaddr, iotexChainID); err != nil {
		return nil, err
	}

	ps := &PubSubs{
		ps:                sub,
		subscribers:       make(map[string]*subscriber),
		selfID:            h.ID(),
		handler:           handler,
		topicEventMonitor: topicEventMonitor,
	}

	go ps.watchTopicEvent()

	return ps, nil
}

type PubSubs struct {
	mux               sync.RWMutex
	subscribers       map[string]*subscriber
	ps                *pubsub.PubSub
	selfID            peer.ID
	handler           P2PDataHandler
	topicEventMonitor TopicEventMonitor
}

func (p *PubSubs) watchTopicEvent() {
	for {
		event := <-p.topicEventMonitor.Subscribe()
		switch event.Type {
		case TopicEventType_Upserted:
			if err := p.add(event.Topic); err != nil {
				slog.Error("failed to add subscriber", "topic", event.Topic)
			}
		case TopicEventType_Paused:
			p.delete(event.Topic)
		}
	}
}

func (p *PubSubs) add(topic string) error {
	p.mux.Lock()
	defer p.mux.Unlock()

	if _, ok := p.subscribers[topic]; ok {
		return nil
	}

	nps, err := newSubscriber(topic, p.ps, p.handler, p.selfID)
	if err != nil {
		return err
	}

	p.subscribers[topic] = nps
	return nil
}

func (p *PubSubs) delete(topic string) {
	p.mux.Lock()
	defer p.mux.Unlock()

	pubSub, ok := p.subscribers[topic]
	if !ok {
		return
	}
	pubSub.release()
	delete(p.subscribers, topic)
}

func (p *PubSubs) get(topic string) (*subscriber, bool) {
	p.mux.RLock()
	defer p.mux.RUnlock()

	s, ok := p.subscribers[topic]
	return s, ok
}

func (p *PubSubs) Publish(topic string, d []byte) error {
	s, ok := p.get(topic)
	if !ok || s == nil {
		slog.Error("topic not exists", "topic", topic)
		return errTopicNotExists
	}

	if err := s.publish(d); err != nil {
		slog.Error("failed to publish", "topic", topic)
		return errFailedToPublish
	}
	return nil
}

var (
	errTopicNotExists         = errors.New("topic not exists")
	errFailedToPublish        = errors.New("failed to publish")
	errFailedToJoinP2P        = errors.New("failed to join p2p networking")
	errFailedToSubscribeTopic = errors.New("failed to subscribe topic")
)
