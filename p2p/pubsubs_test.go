package p2p

import (
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

type mockHost struct{ host.Host }

func (h mockHost) ID() peer.ID {
	return ""
}

type mockTopicEventMonitor struct{}

func (m *mockTopicEventMonitor) Subscribe() <-chan *TopicEvent {
	ch := make(chan *TopicEvent, 10)
	return ch
}

func TestNewPubSubs(t *testing.T) {
	r := require.New(t)

	t.Run("FailedToNewP2PHost", func(t *testing.T) {
		p := gomonkey.NewPatches()
		defer p.Reset()

		p = p.ApplyFuncReturn(libp2p.New, nil, errors.New(t.Name()))

		_, err := NewPubSubs(&mockTopicEventMonitor{}, &mockP2PDataHandler{}, "any", 0)
		r.ErrorContains(err, t.Name())
	})
	t.Run("FailedToNewGossip", func(t *testing.T) {
		p := gomonkey.NewPatches()
		defer p.Reset()

		p = p.ApplyFuncReturn(libp2p.New, &mockHost{}, nil)
		p = p.ApplyFuncReturn(pubsub.NewGossipSub, nil, errors.New(t.Name()))

		_, err := NewPubSubs(&mockTopicEventMonitor{}, &mockP2PDataHandler{}, "any", 0)
		r.ErrorContains(err, t.Name())
	})

	t.Run("FailedToDiscovery", func(t *testing.T) {
		p := gomonkey.NewPatches()
		defer p.Reset()

		p = p.ApplyFuncReturn(libp2p.New, &mockHost{}, nil)
		p = p.ApplyFuncReturn(pubsub.NewGossipSub, &pubsub.PubSub{}, nil)
		p = p.ApplyFuncReturn(discoverPeers, errors.New(t.Name()))

		_, err := NewPubSubs(&mockTopicEventMonitor{}, &mockP2PDataHandler{}, "any", 0)
		r.ErrorContains(err, t.Name())
	})

	t.Run("Success", func(t *testing.T) {
		p := gomonkey.NewPatches()
		defer p.Reset()

		p = p.ApplyFuncReturn(libp2p.New, &mockHost{}, nil)
		p = p.ApplyFuncReturn(pubsub.NewGossipSub, nil, nil)
		p = p.ApplyFuncReturn(discoverPeers, nil)

		_, err := NewPubSubs(&mockTopicEventMonitor{}, &mockP2PDataHandler{}, "any", 0)
		r.NoError(err)
	})
}

func TestPubSubs_add(t *testing.T) {
	r := require.New(t)

	ps := &PubSubs{subscribers: map[string]*subscriber{"exists": {}}}

	t.Run("ProjectIDAlreadyExists", func(t *testing.T) {
		r.Nil(ps.add("exists"))
	})

	t.Run("FailedToNewSubscriber", func(t *testing.T) {
		p := gomonkey.NewPatches()
		defer p.Reset()

		p = p.ApplyFuncReturn(newSubscriber, nil, errors.New(t.Name()))

		err := ps.add("nonexistent1")
		r.EqualError(err, t.Name())
	})

	t.Run("Success", func(t *testing.T) {
		p := gomonkey.NewPatches()
		defer p.Reset()

		p = p.ApplyFuncReturn(newSubscriber, &subscriber{}, nil)

		err := ps.add("nonexistent2")
		r.NoError(err)
	})
}

func TestPubSubs_delete(t *testing.T) {
	ps := &PubSubs{
		subscribers: map[string]*subscriber{"exists": {}},
	}

	t.Run("IDNotExist", func(t *testing.T) {
		ps.delete("nonexistent")
	})

	t.Run("Success", func(t *testing.T) {
		p := gomonkey.NewPatches()
		defer p.Reset()

		p = p.ApplyFuncReturn(newSubscriber, &subscriber{}, nil)
		p = p.ApplyPrivateMethod(&subscriber{}, "release", func(_ *subscriber) {})

		ps.delete("exists")
	})
}

func TestPubSubs_publish(t *testing.T) {
	r := require.New(t)

	ps := &PubSubs{subscribers: map[string]*subscriber{"exists": {}}}
	d := []byte("1")

	t.Run("NotExist", func(t *testing.T) {
		r.Error(ps.Publish("nonexistent", d))
	})

	t.Run("FailedToPublishData", func(t *testing.T) {
		p := gomonkey.NewPatches()
		defer p.Reset()

		p = p.ApplyMethodReturn(&pubsub.Topic{}, "Publish", errors.New(t.Name()))
		r.ErrorContains(ps.Publish("exists", d), errFailedToPublish.Error())
	})

	t.Run("Success", func(t *testing.T) {
		p := gomonkey.NewPatches()
		defer p.Reset()

		p = p.ApplyMethodReturn(&pubsub.Topic{}, "Publish", nil)
		r.NoError(ps.Publish("exists", d))
	})
}
