package project

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path"
	"strconv"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/event"
	"github.com/pkg/errors"

	"github.com/machinefi/sprout/p2p"
	"github.com/machinefi/sprout/project/contracts"
)

func topic(projectID uint64) string {
	return "w3bstream-project-" + strconv.FormatUint(projectID, 10)
}

type Manager struct {
	mux          sync.Mutex
	pool         map[key]*Config
	projectIDs   map[uint64]bool
	instance     *contracts.Contracts
	ipfsEndpoint string
	notify       chan *p2p.TopicEvent
	cache        *cache
	// znodes       []string
	// ioID         string
}

type key string

func getKey(projectID uint64, version string) key {
	return key(fmt.Sprintf("%d_%s", projectID, version))
}

func (m *Manager) Get(projectID uint64, version string) (*Config, error) {
	m.mux.Lock()
	defer m.mux.Unlock()

	if p, ok := m.pool[getKey(projectID, version)]; ok {
		return p, nil
	}
	return nil, errors.Errorf("project config not exist, projectID %d, version %s", projectID, version)
}

func (m *Manager) Set(projectID uint64, version string, c *Config) {
	m.mux.Lock()
	defer m.mux.Unlock()

	key := getKey(projectID, version)
	old, ok := m.pool[key]
	m.pool[key] = c
	if ok {
		slog.Warn("project was overwritten", "project_id", projectID, "version", version, "old_version", old.Version)
	}
	m.projectIDs[projectID] = true
}

// TODO will delete when node konw how to fetch message
func (m *Manager) GetAllProjectID() []uint64 {
	m.mux.Lock()
	defer m.mux.Unlock()

	ids := []uint64{}
	for id := range m.projectIDs {
		ids = append(ids, id)
	}
	return ids
}

func (m *Manager) Subscribe() <-chan *p2p.TopicEvent {
	return m.notify
}

func (m *Manager) doProjectRegistrarWatch(logs <-chan *types.Log, subs event.Subscription) {
	select {
	case err := <-subs.Err():
		slog.Error("project upserted event subscription failed", "err", err)
	case l := <-logs:
		ev, err := m.instance.ParseProjectUpserted(*l)
		if err != nil {
			slog.Error("failed to parse target event", "msg", err)
			return
		}
		if ev.ProjectId == 0 {
			return
		}
		pm := &ProjectMeta{
			ProjectID: ev.ProjectId,
			Uri:       ev.Uri,
			Hash:      ev.Hash,
		}
		data, err := pm.GetConfigData(m.ipfsEndpoint)
		if err != nil {
			slog.Error("failed to fetch project", "error", err, "project_id", ev.ProjectId)
			return
		}
		cs, err := convertConfigs(data)
		if err != nil {
			slog.Error("failed to convert project configs", "error", err, "project_id", ev.ProjectId)
			return
		}
		for _, c := range cs {
			slog.Info("monitor project", "project_id", pm.ProjectID, "version", c.Version, "vm_type", c.VMType, "code_size", len(c.Code))
			m.Set(pm.ProjectID, c.Version, c)
		}
		if m.cache != nil {
			m.cache.set(ev.ProjectId, data)
		}

		if !pm.Paused {
			m.notify <- &p2p.TopicEvent{
				Topic: topic(pm.ProjectID),
				Type:  p2p.TopicEventType_Upserted,
			}
		} else {
			m.notify <- &p2p.TopicEvent{
				Topic: topic(pm.ProjectID),
				Type:  p2p.TopicEventType_Paused,
			}
		}
	}
}

func (m *Manager) watchProjectRegistrar(logs <-chan *types.Log, subs event.Subscription) {
	for {
		m.doProjectRegistrarWatch(logs, subs)
	}
}

func (m *Manager) fillProjectPoolFromContract() {
	for projectID := uint64(1); ; projectID++ {
		emptyHash := [32]byte{}
		mp, err := m.instance.Projects(nil, projectID)
		if err != nil {
			slog.Error("failed to get project meta from chain ", "project_id", projectID, "error", err)
			continue
		}
		// query empty, means reached the maximum projectID value
		if mp.Uri == "" || bytes.Equal(mp.Hash[:], emptyHash[:]) {
			slog.Info("load project from contract completed", "max project_id", projectID-1)
			return
		}

		// znodeMap := map[[sha256.Size]byte]string{}
		// for _, n := range znodes {
		// 	znodeMap[sha256.Sum256([]byte(n))] = n
		// }

		// max := new(big.Int).SetUint64(0)
		// maxZnode := ioID
		// for h, id := range znodeMap {
		// 	n := new(big.Int).Xor(new(big.Int).SetBytes(h[:]), new(big.Int).SetUint64(i))
		// 	if n.Cmp(max) > 0 {
		// 		max = n
		// 		maxZnode = id
		// 	}
		// }
		// if maxZnode != ioID {
		// 	slog.Info("the project not scheduld to this znode", "projectID", i)
		// 	continue
		// }

		pm := &ProjectMeta{
			ProjectID: projectID,
			Uri:       mp.Uri,
			Hash:      mp.Hash,
			Paused:    mp.Paused,
		}

		var data []byte
		cached := true
		if m.cache != nil {
			data = m.cache.get(projectID, mp.Hash[:])
		}
		if len(data) == 0 {
			cached = false
			data, err = pm.GetConfigData(m.ipfsEndpoint)
			if err != nil {
				slog.Error("failed to fetch project", "error", err, "project_id", projectID)
				continue
			}
		}
		cs, err := convertConfigs(data)
		if err != nil {
			slog.Error("failed to convert project config", "error", err, "project_id", projectID)
			continue
		}

		for _, c := range cs {
			slog.Debug("contract project loaded", "project_id", pm.ProjectID, "version", c.Version, "vm_type", c.VMType, "code_size", len(c.Code))
			m.Set(projectID, c.Version, c)
		}
		if !cached && m.cache != nil {
			m.cache.set(projectID, data)
		}
	}
}

func (m *Manager) fillProjectPoolFromLocal(projectFileDir string) {
	if projectFileDir == "" {
		return
	}
	files, err := os.ReadDir(projectFileDir)
	if err != nil {
		if !os.IsNotExist(err) {
			slog.Error("read project directory failed", "path", projectFileDir, "error", err)
			return
		}
	}
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		data, err := os.ReadFile(path.Join(projectFileDir, f.Name()))
		if err != nil {
			slog.Error("read project config failed", "filename", f.Name(), "error", err)
			continue
		}

		projectID, err := strconv.ParseUint(f.Name(), 10, 64)
		if err != nil {
			slog.Error("parse filename failed", "filename", f.Name())
			continue
		}
		cs := []*Config{}
		if err := json.Unmarshal(data, &cs); err != nil {
			slog.Error("parse project config failed", "filename", f.Name())
			continue
		}

		for _, c := range cs {
			slog.Info("local project loaded", "project_id", projectID, "version", c.Version, "vm_type", c.VMType, "code_size", len(c.Code))
			m.Set(projectID, c.Version, c)
		}
	}
}

func NewManager(chainEndpoint, contractAddress, projectFileDir, projectCacheDir, ipfsEndpoint string) (*Manager, error) {
	var c *cache
	var err error
	if projectCacheDir != "" {
		c, err = newCache(projectCacheDir)
		if err != nil {
			return nil, errors.Wrap(err, "failed to new cache")
		}
	}
	m := &Manager{
		pool:         make(map[key]*Config),
		projectIDs:   make(map[uint64]bool),
		ipfsEndpoint: ipfsEndpoint,
		notify:       make(chan *p2p.TopicEvent, 32),
		cache:        c,
	}

	if contractAddress != "" {
		client, err := ethclient.Dial(chainEndpoint)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to dial chain, endpoint %s", chainEndpoint)
		}
		m.instance, err = contracts.NewContracts(common.HexToAddress(contractAddress), client)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to new contract instance, endpoint %s, contractAddress %s", chainEndpoint, contractAddress)
		}

		m.fillProjectPoolFromContract()

		topic := "ProjectUpserted(uint64,string,bytes32)"
		monitor, err := NewDefaultMonitor(
			chainEndpoint,
			[]string{contractAddress},
			[]string{topic},
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to new contract monitor")
		}
		go monitor.run()
		go m.watchProjectRegistrar(monitor.MustEvents(topic), monitor)
	}

	m.fillProjectPoolFromLocal(projectFileDir)

	return m, nil
}
