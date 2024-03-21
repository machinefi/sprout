package p2p

import (
	"context"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func Test_newSubscriber(t *testing.T) {
	r := require.New(t)

	t.Run("FailedToJoinTopic", func(t *testing.T) {
		p := gomonkey.NewPatches()
		defer p.Reset()

		p = p.ApplyMethodReturn(&pubsub.PubSub{}, "Join", nil, errors.New(t.Name()))

		_, err := newSubscriber("any", &pubsub.PubSub{}, nil, peer.ID("0"))
		r.ErrorContains(err, t.Name())
	})

	t.Run("FailedToSubscribeTopic", func(t *testing.T) {
		p := gomonkey.NewPatches()
		defer p.Reset()

		p = p.ApplyMethodReturn(&pubsub.PubSub{}, "Join", &pubsub.Topic{}, nil)
		p = p.ApplyMethodReturn(&pubsub.Topic{}, "Subscribe", nil, errors.New(t.Name()))

		_, err := newSubscriber("any", &pubsub.PubSub{}, nil, peer.ID("0"))
		r.ErrorContains(err, t.Name())
	})

	t.Run("Success", func(t *testing.T) {
		p := gomonkey.NewPatches()
		defer p.Reset()

		p = p.ApplyMethodReturn(&pubsub.PubSub{}, "Join", &pubsub.Topic{}, nil)
		p = p.ApplyMethodReturn(&pubsub.Topic{}, "Subscribe", &pubsub.Subscription{}, nil)

		_, err := newSubscriber("any", &pubsub.PubSub{}, nil, peer.ID("0"))
		r.NoError(err)
	})
}

func Test_subscriber_release(t *testing.T) {
	ps := &subscriber{
		topic:        &pubsub.Topic{},
		cancel:       func() {},
		subscription: &pubsub.Subscription{},
	}

	t.Run("FailedToCloseTopic", func(t *testing.T) {
		p := gomonkey.NewPatches()
		defer p.Reset()

		p = p.ApplyMethod(&pubsub.Subscription{}, "Cancel", func() {})
		p = p.ApplyMethodReturn(&pubsub.Topic{}, "Close", errors.New(t.Name()))

		ps.release()
	})

	t.Run("Success", func(t *testing.T) {
		p := gomonkey.NewPatches()
		defer p.Reset()

		p = p.ApplyMethod(&pubsub.Subscription{}, "Cancel", func() {})
		p = p.ApplyMethodReturn(&pubsub.Topic{}, "Close", nil)

		ps.release()
	})
}

type mockP2PDataHandler struct{}

func (h *mockP2PDataHandler) Handle([]byte) [][]byte {
	return nil
}

func Test_subscriber_run(t *testing.T) {
	p := gomonkey.NewPatches()
	defer p.Reset()

	selfID := peer.ID("self")
	ps := &subscriber{
		selfID:       selfID,
		topic:        &pubsub.Topic{},
		subscription: &pubsub.Subscription{},
		handler:      &mockP2PDataHandler{},
	}

	t.Run("ContextDone", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		ps.run(ctx)
		time.Sleep(time.Second)
	})

	t.Run("SubscribeAndHandleMessage", func(t *testing.T) {
		p = p.ApplyMethodSeq(&pubsub.Subscription{}, "Next", []gomonkey.OutputCell{
			{
				Values: gomonkey.Params{nil, errors.New("any")},
				Times:  1,
			},
			{
				Values: gomonkey.Params{&pubsub.Message{ReceivedFrom: selfID}, nil},
				Times:  1,
			},
			{
				Values: gomonkey.Params{&pubsub.Message{ReceivedFrom: "other"}, nil},
				Times:  5,
			},
		})

		ctx, cancel := context.WithCancel(context.Background())

		go func() {
			time.Sleep(time.Millisecond * 300)
			cancel()
		}()
		go func() {
			defer func() {
				t.Log(recover())
			}()
			ps.run(ctx)
		}()

		time.Sleep(time.Second)
	})
}
