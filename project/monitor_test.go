package project

import (
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestNewMonitor(t *testing.T) {
	r := require.New(t)
	p := gomonkey.NewPatches()
	defer p.Reset()

	t.Run("FailedToDialChain", func(t *testing.T) {
		p = p.ApplyFuncReturn(ethclient.Dial, nil, errors.New(t.Name()))

		_, err := NewMonitor("", []string{}, []string{}, 1, 100, 3*time.Second)
		r.ErrorContains(err, t.Name())
	})
	t.Run("Success", func(t *testing.T) {
		p = p.ApplyFuncReturn(ethclient.Dial, nil, nil)

		_, err := NewMonitor("", []string{"0x02feBE78F3A740b3e9a1CaFAA1b23a2ac0793D26"}, []string{"ProjectUpserted(uint64,string,bytes32)"}, 1, 100, 3*time.Second)
		r.NoError(err)
	})
}

func TestNewDefaultMonitor(t *testing.T) {
	r := require.New(t)
	p := gomonkey.NewPatches()
	defer p.Reset()

	t.Run("FailedToDialChain", func(t *testing.T) {
		p = p.ApplyFuncReturn(ethclient.Dial, nil, errors.New(t.Name()))

		_, err := NewDefaultMonitor("", []string{}, []string{})
		r.ErrorContains(err, t.Name())
	})
	t.Run("FailedToGetBlockNumber", func(t *testing.T) {
		p = p.ApplyFuncReturn(ethclient.Dial, ethclient.NewClient(nil), nil)
		p = p.ApplyMethodReturn(&ethclient.Client{}, "Close")
		p = p.ApplyMethodReturn(&ethclient.Client{}, "BlockNumber", uint64(0), errors.New(t.Name()))

		_, err := NewDefaultMonitor("", []string{}, []string{})
		r.ErrorContains(err, t.Name())
	})
	t.Run("Success", func(t *testing.T) {
		p = p.ApplyMethodReturn(&ethclient.Client{}, "BlockNumber", uint64(100), nil)
		p = p.ApplyFuncReturn(NewMonitor, &Monitor{}, nil)

		_, err := NewDefaultMonitor("", []string{}, []string{})
		r.NoError(err)
	})
}

func TestMonitor_Err(t *testing.T) {
	r := require.New(t)

	t.Run("Success", func(t *testing.T) {
		m := &Monitor{
			err: make(chan error, 1),
		}
		res := m.Err()
		err := errors.New(t.Name())
		m.err <- err
		r.Equal(<-res, err)
	})
}

func TestMonitor_Unsubscribe(t *testing.T) {
	r := require.New(t)

	t.Run("Success", func(t *testing.T) {
		m := &Monitor{
			stop: make(chan struct{}, 1),
		}
		m.Unsubscribe()
		r.NotNil(<-m.stop)
	})
}

func TestMonitor_Events(t *testing.T) {
	r := require.New(t)

	t.Run("Success", func(t *testing.T) {
		topic := "test"
		m := &Monitor{
			events: map[common.Hash]chan *types.Log{crypto.Keccak256Hash([]byte(topic)): make(chan *types.Log)},
		}
		_, ok := m.Events(topic)
		r.True(ok)
	})
}

func TestMonitor_MustEvents(t *testing.T) {
	r := require.New(t)

	t.Run("Panic", func(t *testing.T) {
		defer func() {
			err := recover()
			r.NotNil(err)
		}()

		topic := "test"
		m := &Monitor{
			events: map[common.Hash]chan *types.Log{},
		}
		_ = m.MustEvents(topic)
	})

	t.Run("Success", func(t *testing.T) {
		topic := "test"
		m := &Monitor{
			events: map[common.Hash]chan *types.Log{crypto.Keccak256Hash([]byte(topic)): make(chan *types.Log)},
		}
		ch := m.MustEvents(topic)
		r.NotNil(ch)
	})
}

func TestMonitor_monitorEvent(t *testing.T) {
	r := require.New(t)

	t.Run("QueryLatestBlockNumber", func(t *testing.T) {
		t.Run("FailedToQueryLatestBlockNumber", func(t *testing.T) {
			p := gomonkey.NewPatches()
			defer p.Reset()

			m := &Monitor{}

			p = p.ApplyMethodReturn(&ethclient.Client{}, "BlockNumber", uint64(0), errors.New(t.Name()))
			queried, err := m.monitorEvent()

			r.Equal(queried, 0)
			r.ErrorContains(err, t.Name())
		})
	})

	t.Run("CheckIfNeedQuery", func(t *testing.T) {
		t.Run("AlreadyQueriedTheLatestBlock", func(t *testing.T) {
			p := gomonkey.NewPatches()
			defer p.Reset()

			m := &Monitor{latest: 100}

			p = p.ApplyMethodReturn(&ethclient.Client{}, "BlockNumber", uint64(100), nil)
			queried, err := m.monitorEvent()

			r.Equal(queried, 0)
			r.NoError(err)
		})
	})

	t.Run("BuildQueryAndFilterLogs", func(t *testing.T) {
		t.Run("FailedToFilterLogs", func(t *testing.T) {
			p := gomonkey.NewPatches()
			defer p.Reset()

			m := &Monitor{latest: 99, step: 10000}

			p = p.ApplyMethodReturn(&ethclient.Client{}, "BlockNumber", uint64(100), nil)
			p = p.ApplyMethodReturn(&ethclient.Client{}, "FilterLogs", nil, errors.New(t.Name()))
			queried, err := m.monitorEvent()

			r.Equal(queried, 0)
			r.ErrorContains(err, t.Name())
		})
	})

	t.Run("UpdateTheLatestBlockFlagAndDispatchLogsByTopic", func(t *testing.T) {
		p := gomonkey.NewPatches()
		defer p.Reset()

		topicexists := crypto.Keccak256Hash([]byte("0"))
		topicnonexistent := crypto.Keccak256Hash([]byte("1"))
		m := &Monitor{
			latest: 99,
			step:   10000,
			events: map[common.Hash]chan *types.Log{
				topicexists: make(chan *types.Log, 1),
			},
		}

		p = p.ApplyMethodReturn(&ethclient.Client{}, "BlockNumber", uint64(101), nil)
		p = p.ApplyMethodReturn(&ethclient.Client{}, "FilterLogs", []types.Log{
			{Topics: []common.Hash{topicexists}},
			{Topics: []common.Hash{topicnonexistent}},
		}, nil)

		queried, err := m.monitorEvent()
		r.Equal(queried, 2)
		r.NoError(err)
		r.Equal(m.latest, int64(101))
	})
}

func TestMonitor_run(t *testing.T) {
	p := gomonkey.NewPatches()
	defer p.Reset()

	m := &Monitor{interval: time.Millisecond * 500, stop: make(chan struct{}, 1)}

	p.ApplyPrivateMethod(&Monitor{}, "monitorEvent", func(_ *Monitor) (int, error) { return 0, errors.New("any") })

	go m.run()
	time.Sleep(time.Second)
	m.Unsubscribe()
	time.Sleep(time.Second)

	p.ApplyPrivateMethod(&Monitor{}, "monitorEvent", func(_ *Monitor) (int, error) { return 1, nil })

	go m.run()
	time.Sleep(time.Second)
	m.Unsubscribe()
	time.Sleep(time.Second)
}
