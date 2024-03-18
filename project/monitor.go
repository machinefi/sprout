package project

import (
	"context"
	"log/slog"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/event"
	"github.com/pkg/errors"
)

func NewMonitor(chainEndpoint string, addresses []string, topics []string, from, step int64, interval time.Duration) (*Monitor, error) {
	client, err := ethclient.Dial(chainEndpoint)
	if err != nil {
		return nil, errors.Wrapf(err, "dial chain endpoint %s failed", chainEndpoint)
	}

	m := &Monitor{
		client:   client,
		latest:   from,
		step:     step,
		interval: interval,
		events:   nil,
		err:      make(chan error),
		stop:     make(chan struct{}),
	}
	m.addresses = make([]common.Address, 0, len(addresses))
	for _, addr := range addresses {
		m.addresses = append(m.addresses, common.HexToAddress(addr))
	}

	m.topics = make([][]common.Hash, 1)
	m.events = make(map[common.Hash]chan *types.Log)
	for i := range topics {
		topic := crypto.Keccak256Hash([]byte(topics[i]))
		m.topics[0] = append(m.topics[0], topic)
		m.events[topic] = make(chan *types.Log, 100)
	}

	return m, nil
}

func NewDefaultMonitor(chainEndpoint string, addresses []string, topics []string) (*Monitor, error) {
	client, err := ethclient.Dial(chainEndpoint)
	if err != nil {
		return nil, errors.Wrapf(err, "dial chain endpoint %s failed", chainEndpoint)
	}
	defer client.Close()

	latestBlk, err := client.BlockNumber(context.Background())
	if err != nil {
		return nil, errors.Wrap(err, "get current block number failed")
	}

	return NewMonitor(chainEndpoint, addresses, topics, int64(latestBlk), 100000, time.Second*10)
}

type Monitor struct {
	// client chain client
	client *ethclient.Client
	// latest queried block number
	latest int64
	// step query step, before chased the latest block, query [current, current+step]
	step int64
	// interval query/monitor interval
	interval time.Duration
	// addresses contract addresses
	addresses []common.Address
	// topics care about
	topics [][]common.Hash
	// events channel
	events map[common.Hash]chan *types.Log
	// err failed signal
	err chan error
	// stop signal
	stop chan struct{}
}

var _ event.Subscription = (*Monitor)(nil)

func (m *Monitor) Err() <-chan error {
	return m.err
}

func (m *Monitor) Unsubscribe() {
	m.stop <- struct{}{}
}

func (m *Monitor) Events(topic string) (<-chan *types.Log, bool) {
	ch, ok := m.events[crypto.Keccak256Hash([]byte(topic))]
	return ch, ok
}

func (m *Monitor) MustEvents(topic string) <-chan *types.Log {
	ch, ok := m.Events(topic)
	if !ok {
		panic("event not subscribed " + topic)
	}
	return ch
}

func (m *Monitor) monitorEvent() (int, error) {
	latestBlk, err := m.client.BlockNumber(context.Background())
	if err != nil {
		slog.Error("failed to query latest block number", "error", err)
		return 0, err
	}
	slog.Debug("", "latest block number", latestBlk)
	if uint64(m.latest) >= latestBlk {
		return 0, nil
	}

	query := ethereum.FilterQuery{
		FromBlock: big.NewInt(m.latest),
		ToBlock:   big.NewInt(min(m.latest+m.step, int64(latestBlk))),
		Addresses: m.addresses,
		Topics:    m.topics,
	}

	logs, err := m.client.FilterLogs(context.Background(), query)
	if err != nil {
		slog.Debug("failed to filter logs", "error", err)
		return 0, err
	}

	m.latest = query.ToBlock.Int64()

	for i := range logs {
		topic := logs[i].Topics[0]
		if _, ok := m.events[topic]; !ok {
			continue
		}
		m.events[topic] <- &logs[i]
	}
	return len(logs), nil
}

func (m *Monitor) run() {
	for {
		select {
		case <-m.stop:
			slog.Info("monitor stopped")
			return
		default:
			queried, err := m.monitorEvent()
			if err != nil {
				slog.Error("failed to monitor contract event", "error", err)
			}
			slog.Info("monitor contract event", "queried", queried, "latest", m.latest)
			if queried == 0 {
				time.Sleep(m.interval)
			}
		}
	}
}
