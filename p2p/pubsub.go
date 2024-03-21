package p2p

import (
	"context"
	"log/slog"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
)

func newSubscriber(topic string, ps *pubsub.PubSub, handler P2PDataHandler, selfID peer.ID) (*subscriber, error) {
	l := slog.With("topic", topic)

	_topic, err := ps.Join(topic)
	if err != nil {
		l.Error(err.Error())
		return nil, errors.Wrap(err, errFailedToJoinP2P.Error())
	}
	sub, err := _topic.Subscribe()
	if err != nil {
		l.Error(err.Error())
		return nil, errors.Wrap(err, errFailedToSubscribeTopic.Error())
	}
	ctx, cancel := context.WithCancel(context.Background())

	_ps := &subscriber{
		selfID:       selfID,
		topic:        _topic,
		subscription: sub,
		handler:      handler,
		cancel:       cancel,
	}

	go _ps.run(ctx)
	l.Info("subscribing started")

	return _ps, nil
}

type subscriber struct {
	selfID       peer.ID
	topic        *pubsub.Topic
	subscription *pubsub.Subscription
	handler      P2PDataHandler
	cancel       context.CancelFunc
}

func (p *subscriber) release() {
	p.subscription.Cancel()
	p.cancel()
	if err := p.topic.Close(); err != nil {
		slog.Error("failed to close topic", "error", err, "topic", p.topic.String())
	}
}

func (p *subscriber) publish(data []byte) error {
	return p.topic.Publish(context.Background(), data)
}

func (p *subscriber) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			slog.With("ctx.Err()", ctx.Err()).Info("pubsub stopped caused by")
			return
		default:
			m, err := p.subscription.Next(ctx)
			if err != nil {
				slog.Error("failed to get p2p data", "error", err)
				continue
			}
			if m.ReceivedFrom == p.selfID {
				slog.Info("skip message from self")
				continue
			}
			outputs := p.handler.Handle(m.Message.Data)
			for _, output := range outputs {
				if err := p.topic.Publish(context.Background(), output); err != nil {
					slog.Error("failed to publish output", "error", err)
				}
			}
		}
	}
}
