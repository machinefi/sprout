package task

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/pkg/errors"

	"github.com/machinefi/sprout/p2p"
	"github.com/machinefi/sprout/project"
	"github.com/machinefi/sprout/types"
	"github.com/machinefi/sprout/vm"
)

type Processor struct {
	vmHandler      *vm.Handler
	projectManager *project.Manager
	ps             *p2p.PubSubs
}

func (r *Processor) handleP2PData(d *p2p.Data, topic *pubsub.Topic) {
	if d.Task == nil {
		return
	}
	t := d.Task
	slog.Debug("get a new task", "task_id", t.ID)
	r.reportSuccess(t, types.TaskStateDispatched, nil, topic)

	config, err := r.projectManager.Get(t.ProjectID, t.ProjectVersion)
	if err != nil {
		slog.Error("failed to get project", "error", err, "project_id", t.ProjectID, "project_version", t.ProjectVersion)
		r.reportFail(t, err, topic)
		return
	}

	res, err := r.vmHandler.Handle(t, config.VMType, config.Code, config.CodeExpParam)
	if err != nil {
		slog.Error("failed to generate proof", "error", err)
		r.reportFail(t, err, topic)
		return
	}
	r.reportSuccess(t, types.TaskStateProved, res, topic)
}

func (r *Processor) reportFail(t *types.Task, err error, topic *pubsub.Topic) {
	j, err := json.Marshal(&p2p.Data{
		TaskStateLog: &types.TaskStateLog{
			Task:      *t,
			State:     types.TaskStateFailed,
			Comment:   err.Error(),
			CreatedAt: time.Now(),
		},
	})
	if err != nil {
		slog.Error("failed to marshal p2p task state log data to json", "error", err, "task_id", t.ID)
		return
	}
	if err := topic.Publish(context.Background(), j); err != nil {
		slog.Error("failed to publish task state log data to p2p network", "error", err, "task_id", t.ID)
	}
}

func (r *Processor) reportSuccess(t *types.Task, state types.TaskState, result []byte, topic *pubsub.Topic) {
	j, err := json.Marshal(&p2p.Data{
		TaskStateLog: &types.TaskStateLog{
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
	if err := topic.Publish(context.Background(), j); err != nil {
		slog.Error("failed to publish task state log data to p2p network", "error", err, "task_id", t.ID)
	}
}

func (r *Processor) Run() {
	// TODO project load & delete
}

func (r *Processor) monitorProjectRegistrar(notifier <-chan uint64) {
	for {
		select {
		case projectID := <-notifier:
			if err := r.ps.Add(projectID); err != nil {
				slog.Error("add project pubsub failed", "projectID", projectID, "error", err)
				continue
			}
			slog.Debug("processor project added", "projectID", projectID)
		}
	}
}

func NewProcessor(vmHandler *vm.Handler, projectManager *project.Manager, bootNodeMultiaddr string, iotexChainID int) (*Processor, error) {
	p := &Processor{
		vmHandler:      vmHandler,
		projectManager: projectManager,
	}

	ps, err := p2p.NewPubSubs(p.handleP2PData, bootNodeMultiaddr, iotexChainID)
	if err != nil {
		return nil, err
	}
	p.ps = ps

	for _, id := range projectManager.GetAllProjectID() {
		if err := ps.Add(id); err != nil {
			return nil, errors.Wrapf(err, "add project %d pubsub failed", id)
		}
		slog.Debug("processor project added", "projectID", id)
	}

	go p.monitorProjectRegistrar(projectManager.GetNotify())

	return p, nil
}
