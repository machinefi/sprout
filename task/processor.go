package task

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"

	"github.com/machinefi/sprout/utils/distance"
	"github.com/machinefi/sprout/vm"
)

type VMHandler interface {
	Handle(projectID uint64, vmtype vm.Type, code string, expParam string, data [][]byte) ([]byte, error)
}

type Processor struct {
	vmHandler            VMHandler
	projectConfigManager ProjectConfigManager
	proverID             string
	projectProvers       sync.Map
}

func (r *Processor) HandleProjectProvers(projectID uint64, provers []string) {
	r.projectProvers.Store(projectID, provers)
}

func (r *Processor) HandleP2PData(data []byte, topic *pubsub.Topic) {
	d := p2pData{}
	if err := json.Unmarshal(data, &d); err != nil {
		slog.Error("failed to unmarshal p2p data", "error", err)
		return
	}
	if d.Task == nil {
		return
	}

	t := d.Task

	p, err := r.projectConfigManager.Get(t.ProjectID, t.ProjectVersion)
	if err != nil {
		slog.Error("failed to get project", "error", err, "project_id", t.ProjectID, "project_version", t.ProjectVersion)
		r.reportFail(t, err, topic)
		return
	}

	var provers []string
	proversValue, ok := r.projectProvers.Load(t.ProjectID)
	if ok {
		provers = proversValue.([]string)
	}
	if len(provers) > 1 {
		workProver := distance.GetMinNLocation(provers, t.ID, 1)
		if workProver[0] != r.proverID {
			slog.Info("the task not scheduld to this prover", "project_id", t.ProjectID, "task_id", t.ID)
			return
		}
	}

	slog.Debug("get a new task", "task_id", t.ID)
	r.reportSuccess(t, TaskStateDispatched, nil, topic)

	res, err := r.vmHandler.Handle(t.ProjectID, p.VMType, p.Code, p.CodeExpParam, t.Data)
	if err != nil {
		slog.Error("failed to generate proof", "error", err)
		r.reportFail(t, err, topic)
		return
	}
	r.reportSuccess(t, TaskStateProved, res, topic)
}

func (r *Processor) reportFail(t *Task, err error, topic *pubsub.Topic) {
	d, err := json.Marshal(&p2pData{
		TaskStateLog: &TaskStateLog{
			Task:      *t,
			State:     TaskStateFailed,
			Comment:   err.Error(),
			CreatedAt: time.Now(),
		},
	})
	if err != nil {
		slog.Error("failed to marshal p2p task state log data to json", "error", err, "task_id", t.ID)
		return
	}
	if err := topic.Publish(context.Background(), d); err != nil {
		slog.Error("failed to publish task state log data to p2p network", "error", err, "task_id", t.ID)
	}
}

func (r *Processor) reportSuccess(t *Task, state TaskState, result []byte, topic *pubsub.Topic) {
	d, err := json.Marshal(&p2pData{
		TaskStateLog: &TaskStateLog{
			Task:      *t,
			State:     state,
			Result:    result,
			CreatedAt: time.Now(),
		},
	})
	if err != nil {
		slog.Error("failed to marshal p2p task state log data to json", "error", err, "task_id", t.ID)
		return
	}
	if err := topic.Publish(context.Background(), d); err != nil {
		slog.Error("failed to publish task state log data to p2p network", "error", err, "task_id", t.ID)
	}
}

func NewProcessor(vmHandler VMHandler, projectConfigManager ProjectConfigManager, proverID string) *Processor {
	return &Processor{
		vmHandler:            vmHandler,
		projectConfigManager: projectConfigManager,
		proverID:             proverID,
	}
}
