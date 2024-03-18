package project

import (
	"context"
	"encoding/json"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/machinefi/sprout/project/contracts"
	wstypes "github.com/machinefi/sprout/types"
)

func TestNewManager(t *testing.T) {
	r := require.New(t)

	t.Run("FillProjectFromContractIfContractAddressIsValid", func(t *testing.T) {
		t.Run("EmptyContractAddress", func(t *testing.T) {
			_, err := NewManager("", "", "", "")
			r.Nil(err)
		})

		t.Run("ValidContractAddress", func(t *testing.T) {
			t.Run("DailEth", func(t *testing.T) {
				t.Run("FailedToDialChain", func(t *testing.T) {
					p := gomonkey.NewPatches()
					defer p.Reset()

					p = p.ApplyFuncReturn(ethclient.Dial, nil, errors.New(t.Name()))

					_, err := NewManager("any", "any", "any", "any")
					r.ErrorContains(err, t.Name())
				})
			})

			t.Run("NewContractInstance", func(t *testing.T) {
				p := gomonkey.NewPatches()
				defer p.Reset()

				p = p.ApplyFuncReturn(ethclient.Dial, ethclient.NewClient(&rpc.Client{}), nil)

				t.Run("FailedToNewContracts", func(t *testing.T) {
					p = p.ApplyFuncReturn(contracts.NewContracts, nil, errors.New(t.Name()))

					_, err := NewManager("any", "any", "any", "any")
					r.ErrorContains(err, t.Name())
				})
			})

			t.Run("FillProjectFromContract", func(t *testing.T) {
				p := gomonkey.NewPatches()
				defer p.Reset()

				p = p.ApplyFuncReturn(ethclient.Dial, ethclient.NewClient(&rpc.Client{}), nil)
				p = p.ApplyFuncReturn(contracts.NewContracts, &contracts.Contracts{}, nil)
				p = p.ApplyPrivateMethod(&Manager{}, "fillProjectPoolFromContract", func(_ *Manager) {})

				t.Run("FailedToNewContractMonitor", func(t *testing.T) {
					p = p.ApplyFuncReturn(NewDefaultMonitor, nil, errors.New(t.Name()))

					_, err := NewManager("any", "any", "any", "any")
					r.ErrorContains(err, t.Name())
				})

				t.Run("StartMonitorAndWatchProjectRegistrar", func(t *testing.T) {
					t.Skipf("skip because go routine cannot mock")
					p = p.ApplyFuncReturn(NewDefaultMonitor, &Monitor{}, nil)
					p = p.ApplyPrivateMethod(&Monitor{}, "run", func(_ *Monitor) {
						time.Sleep(time.Second)
						return
					})
					p = p.ApplyPrivateMethod(&Manager{}, "watchProjectRegistrar", func(_ *Monitor, _ <-chan *types.Log, subscription event.Subscription) {
						time.Sleep(time.Second)
						return
					})

					_, err := NewManager("any", "any", "any", "any")
					r.ErrorContains(err, t.Name())
				})
			})
		})
	})
}

func TestManager_Get(t *testing.T) {
	r := require.New(t)

	t.Run("NotExist", func(t *testing.T) {
		m := &Manager{}
		_, err := m.Get(1, "0.1")
		r.ErrorContains(err, "project config not exist")
	})
	t.Run("Success", func(t *testing.T) {
		m := &Manager{
			pool: map[key]*Config{getKey(1, "0.1"): {}},
		}
		_, err := m.Get(1, "0.1")
		r.NoError(err)
	})
}

func TestManager_Set(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		m := &Manager{
			pool: map[key]*Config{
				getKey(1, "0.1"): {},
				getKey(2, "0.2"): {},
			},
			projectIDs: map[uint64]bool{},
		}
		m.Set(1, "0.1", &Config{})
		m.Set(2, "0.2", &Config{})
	})
}

func TestManager_GetAllProjectID(t *testing.T) {
	r := require.New(t)

	t.Run("Success", func(t *testing.T) {
		m := &Manager{
			projectIDs: map[uint64]bool{1: true},
		}
		ids := m.GetAllProjectID()
		r.Equal(len(ids), 1)
		r.Equal(ids[0], uint64(1))
	})
}

func TestManager_GetNotify(t *testing.T) {
	r := require.New(t)

	t.Run("Success", func(t *testing.T) {
		m := &Manager{
			projectIDs: map[uint64]bool{1: true},
			notify:     make(chan uint64, 1),
		}
		notify := m.GetNotify()
		m.notify <- uint64(1)
		d := <-notify
		r.Equal(d, uint64(1))
	})
}

func TestManager_onContractEvent(t *testing.T) {
	r := require.New(t)

	m := &Manager{
		mux:          sync.Mutex{},
		pool:         make(map[key]*Config),
		projectIDs:   make(map[uint64]bool),
		instance:     &contracts.Contracts{},
		ipfsEndpoint: "any",
		notify:       make(chan uint64, 1),
	}

	t.Run("ParseContractLog", func(t *testing.T) {
		t.Run("FailedToParseProjectUpsertedLog", func(t *testing.T) {
			p := gomonkey.NewPatches()
			defer p.Reset()

			p = p.ApplyMethodReturn(&contracts.ContractsFilterer{}, "ParseProjectUpserted", nil, errors.New(t.Name()))

			err := m.onContractEvent(&types.Log{})
			r.ErrorContains(err, t.Name())
		})
	})

	t.Run("CheckEventValid", func(t *testing.T) {
		t.Run("InvalidProjectID", func(t *testing.T) {
			p := gomonkey.NewPatches()
			defer p.Reset()

			p = p.ApplyMethodReturn(&contracts.ContractsFilterer{}, "ParseProjectUpserted", &contracts.ContractsProjectUpserted{
				ProjectId: 0,
			}, nil)

			err := m.onContractEvent(&types.Log{})
			r.Equal(err, ErrInvalidProjectID)
		})
	})

	t.Run("FetchProject", func(t *testing.T) {
		t.Run("FailedToGetProjectConfig", func(t *testing.T) {
			p := gomonkey.NewPatches()
			defer p.Reset()

			p = p.ApplyMethodReturn(&contracts.ContractsFilterer{}, "ParseProjectUpserted", &contracts.ContractsProjectUpserted{
				ProjectId: 1,
				Uri:       "any",
				Hash:      [32]byte{},
			}, nil)
			p = p.ApplyMethodReturn(&ProjectMeta{}, "GetConfigs", nil, errors.New(t.Name()))

			err := m.onContractEvent(&types.Log{})
			r.ErrorContains(err, t.Name())
		})
	})

	t.Run("UpdateProjectPool", func(t *testing.T) {
		p := gomonkey.NewPatches()
		defer p.Reset()

		p = p.ApplyMethodReturn(&contracts.ContractsFilterer{}, "ParseProjectUpserted", &contracts.ContractsProjectUpserted{
			ProjectId: 100,
			Uri:       "any",
			Hash:      [32]byte{},
		}, nil)
		p = p.ApplyMethodReturn(&ProjectMeta{}, "GetConfigs", []*Config{
			{
				Code:         "any",
				CodeExpParam: "any",
				VMType:       "any",
				Output:       OutputConfig{},
				Aggregation:  AggregationConfig{},
				Version:      "0.1",
			},
		}, nil)

		err := m.onContractEvent(&types.Log{})
		r.NoError(err)
		nc := m.GetNotify()
		r.Equal(len(nc), 1)
		r.Equal(<-nc, uint64(100))
	})
}

func newMockSubscription() *mockSubscription {
	sub := &mockSubscription{
		errChain: make(chan error),
	}
	sub.ctx, sub.cancel = context.WithCancel(context.Background())
	return sub
}

type mockSubscription struct {
	errChain chan error
	ctx      context.Context
	cancel   context.CancelFunc
}

func (s *mockSubscription) Err() <-chan error {
	return s.errChain
}

func (s *mockSubscription) Unsubscribe() {
	s.cancel()
	s.errChain <- s.ctx.Err()
}

func TestManager_watchProjectRegistrar(t *testing.T) {
	r := require.New(t)

	p := gomonkey.NewPatches()
	defer p.Reset()

	m := &Manager{}

	times := -1
	p = p.ApplyPrivateMethod(&Manager{}, "onContractEvent", func(_ *Manager, _ *types.Log) error {
		times++
		if times%2 == 0 {
			return errors.New("any")
		}
		return nil
	})

	logs := make(chan *types.Log, 1)
	subs := newMockSubscription()

	var err error
	go func() {
		err = m.watchProjectRegistrar(logs, subs)
	}()

	logs <- &types.Log{}
	time.Sleep(time.Millisecond * 200)
	logs <- &types.Log{}
	time.Sleep(time.Millisecond * 200)
	subs.Unsubscribe()
	time.Sleep(time.Millisecond * 200)

	r.Error(err)
}

func TestManager_fillProjectPoolFromContract(t *testing.T) {
	r := require.New(t)
	p := gomonkey.NewPatches()
	defer p.Reset()

	t.Run("FailedToReadChain", func(t *testing.T) {
		outputs := []gomonkey.OutputCell{
			{
				Values: gomonkey.Params{struct {
					Uri    string
					Hash   [32]byte
					Paused bool
				}{}, errors.New(t.Name())},
			},
			{
				Values: gomonkey.Params{struct {
					Uri    string
					Hash   [32]byte
					Paused bool
				}{}, nil},
			},
		}
		p = p.ApplyMethodSeq(&contracts.ContractsCaller{}, "Projects", outputs)

		m := &Manager{
			projectIDs: map[uint64]bool{},
			pool:       map[key]*Config{},
			instance:   &contracts.Contracts{},
		}
		m.fillProjectPoolFromContract()
		r.Equal(len(m.GetAllProjectID()), 0)
	})
	t.Run("Finished", func(t *testing.T) {
		p = p.ApplyMethodReturn(&contracts.ContractsCaller{}, "Projects", struct {
			Uri    string
			Hash   [32]byte
			Paused bool
		}{}, nil)

		m := &Manager{
			projectIDs: map[uint64]bool{},
			pool:       map[key]*Config{},
			instance:   &contracts.Contracts{},
		}
		m.fillProjectPoolFromContract()
		r.Equal(len(m.GetAllProjectID()), 0)
	})
	t.Run("FailedToGetConfig", func(t *testing.T) {
		outputs := []gomonkey.OutputCell{
			{
				Values: gomonkey.Params{struct {
					Uri    string
					Hash   [32]byte
					Paused bool
				}{
					Uri:  "test",
					Hash: [32]byte{1},
				}, nil},
			},
			{
				Values: gomonkey.Params{struct {
					Uri    string
					Hash   [32]byte
					Paused bool
				}{}, nil},
			},
		}
		p = p.ApplyMethodSeq(&contracts.ContractsCaller{}, "Projects", outputs)
		p = p.ApplyMethodReturn(&ProjectMeta{}, "GetConfigs", nil, errors.New(t.Name()))

		m := &Manager{
			projectIDs: map[uint64]bool{},
			pool:       map[key]*Config{},
			instance:   &contracts.Contracts{},
		}
		m.fillProjectPoolFromContract()
		r.Equal(len(m.GetAllProjectID()), 0)
	})
	t.Run("Success", func(t *testing.T) {
		outputs := []gomonkey.OutputCell{
			{
				Values: gomonkey.Params{struct {
					Uri    string
					Hash   [32]byte
					Paused bool
				}{
					Uri:  "test",
					Hash: [32]byte{1},
				}, nil},
			},
			{
				Values: gomonkey.Params{struct {
					Uri    string
					Hash   [32]byte
					Paused bool
				}{}, nil},
			},
		}
		p = p.ApplyMethodSeq(&contracts.ContractsCaller{}, "Projects", outputs)
		p = p.ApplyMethodReturn(&ProjectMeta{}, "GetConfigs", []*Config{{}}, nil)

		m := &Manager{
			projectIDs: map[uint64]bool{},
			pool:       map[key]*Config{},
			instance:   &contracts.Contracts{},
		}
		m.fillProjectPoolFromContract()
		r.Equal(len(m.GetAllProjectID()), 1)
	})
}

type mockDirEntry struct {
	isDir bool
	name  string
}

func (m mockDirEntry) Name() string {
	return m.name
}

func (m mockDirEntry) IsDir() bool {
	return m.isDir
}

func (m mockDirEntry) Type() os.FileMode {
	return 0
}

func (m mockDirEntry) Info() (os.FileInfo, error) {
	return nil, nil
}

func TestManager_fillProjectPoolFromLocal(t *testing.T) {
	r := require.New(t)
	p := gomonkey.NewPatches()
	defer p.Reset()

	cs := []*Config{
		{
			Code:    "i am code",
			VMType:  wstypes.VMHalo2,
			Version: "0.1",
		},
	}
	jc, err := json.Marshal(cs)
	r.NoError(err)

	t.Run("ProjectFileDirParamIsEmpty", func(t *testing.T) {
		m := &Manager{
			projectIDs: map[uint64]bool{},
			pool:       map[key]*Config{},
		}
		m.fillProjectPoolFromLocal("")
		r.Equal(len(m.GetAllProjectID()), 0)
	})
	t.Run("ReadDirNotExistError", func(t *testing.T) {
		p = p.ApplyFuncReturn(os.ReadDir, nil, os.ErrNotExist)

		m := &Manager{
			projectIDs: map[uint64]bool{},
			pool:       map[key]*Config{},
		}
		m.fillProjectPoolFromLocal("test")
		r.Equal(len(m.GetAllProjectID()), 0)
	})
	t.Run("FailedToReadDir", func(t *testing.T) {
		p = p.ApplyFuncReturn(os.ReadDir, nil, errors.New(t.Name()))

		m := &Manager{
			projectIDs: map[uint64]bool{},
			pool:       map[key]*Config{},
		}
		m.fillProjectPoolFromLocal("test")
		r.Equal(len(m.GetAllProjectID()), 0)
	})
	t.Run("FailedToReadFile", func(t *testing.T) {
		p = p.ApplyFuncReturn(os.ReadDir, []os.DirEntry{mockDirEntry{isDir: true}, mockDirEntry{}}, nil)
		p = p.ApplyFuncReturn(os.ReadFile, nil, errors.New(t.Name()))

		m := &Manager{
			projectIDs: map[uint64]bool{},
			pool:       map[key]*Config{},
		}
		m.fillProjectPoolFromLocal("test")
		r.Equal(len(m.GetAllProjectID()), 0)
	})
	t.Run("InvalidFileName", func(t *testing.T) {
		p = p.ApplyFuncReturn(os.ReadFile, nil, nil)

		m := &Manager{
			projectIDs: map[uint64]bool{},
			pool:       map[key]*Config{},
		}
		m.fillProjectPoolFromLocal("test")
		r.Equal(len(m.GetAllProjectID()), 0)
	})
	t.Run("Success", func(t *testing.T) {
		p = p.ApplyFuncReturn(os.ReadDir, []os.DirEntry{mockDirEntry{isDir: true}, mockDirEntry{name: "1"}}, nil)
		p = p.ApplyFuncReturn(os.ReadFile, jc, nil)

		m := &Manager{
			projectIDs: map[uint64]bool{},
			pool:       map[key]*Config{},
		}
		m.fillProjectPoolFromLocal("test")
		r.Equal(len(m.GetAllProjectID()), 1)
	})
	t.Run("FailedToUnmarshalJson", func(t *testing.T) {
		p = p.ApplyFuncReturn(json.Unmarshal, errors.New(t.Name()))

		m := &Manager{
			projectIDs: map[uint64]bool{},
			pool:       map[key]*Config{},
		}
		m.fillProjectPoolFromLocal("test")
		r.Equal(len(m.GetAllProjectID()), 0)
	})
}
