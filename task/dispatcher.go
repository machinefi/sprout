package task

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/pkg/errors"

	"github.com/machinefi/sprout/output"
)

func NewDispatcher(persistence Persistence, projects ProjectPool, operatorPrivateKey, operatorPrivateKeyED25519 string) (*Dispatcher, error) {
	d := &Dispatcher{
		persistence:               persistence,
		projects:                  projects,
		operatorPrivateKeyECDSA:   operatorPrivateKey,
		operatorPrivateKeyED25519: operatorPrivateKeyED25519,
	}

	return d, nil
}

type Dispatcher struct {
	persistence               Persistence
	projects                  ProjectPool
	operatorPrivateKeyECDSA   string
	operatorPrivateKeyED25519 string
}

func (d *Dispatcher) Handle(input []byte) (outputs [][]byte) {
	data := &p2pData{}
	if err := data.Unmarshal(input); err != nil {
		slog.Error("failed to unmarshal p2p data", "error", err)
		return nil
	}

	if data.TaskStateLog == nil {
		return nil
	}

	l := data.TaskStateLog
	r := &TaskStateLog{Task: l.Task}
	if err := d.persistence.Create(l); err != nil {
		slog.Error("failed to create task state log", "error", err, "taskID", l.Task.ID)
		return nil
	}

	if l.State != TaskStateProved {
		return nil
	}

	config, err := d.projects.Get(l.Task.ProjectID, l.Task.ProjectVersion)
	if err != nil {
		slog.Error("failed to get project", "error", err, "project_id", l.Task.ProjectID, "project_version", l.Task.ProjectVersion)
		return nil
	}

	defer func() {
		r.CreatedAt = time.Now()
		if err = d.persistence.Create(r); err != nil {
			slog.Error("failed to create outputted task state", "error", err, "taskID", l.Task.ID)
		}
	}()

	output, err := output.New(&config.Output, d.operatorPrivateKeyECDSA, d.operatorPrivateKeyED25519)
	if err != nil {
		slog.Error("failed to init output", "error", err)
		r.State = TaskStateFailed
		r.Comment = err.Error()
		return
	}

	slog.Debug("output proof", "outputter", fmt.Sprintf("%T", output))

	result, err := output.Output(l.Task.ProjectID, l.Task.Data, l.Result)
	if err != nil {
		slog.Error("failed to output", "error", err, "taskID", l.Task.ID)
		r.State = TaskStateFailed
		r.Comment = err.Error()
		return
	}

	r.State = TaskStateOutputted
	r.Result = []byte(result)
	return
}

func Dispatching(id uint64, datasource Datasource, networking Networking) {
	ticker := time.NewTicker(3 * time.Second)

	for range ticker.C {
		next, err := dispatch(id, datasource, networking)
		if err != nil {
			slog.Error("failed to dispatch task", "error", err)
			continue
		}
		id = next
	}
}

func dispatch(taskID uint64, datasource Datasource, networking Networking) (uint64, error) {
	t, err := datasource.Retrieve(taskID)
	if err != nil {
		return 0, errors.Wrap(err, "failed to retrieve task from data source")
	}

	if t == nil {
		return taskID, nil
	}

	data, err := (&p2pData{Task: t}).Marshal()
	if err != nil {
		return 0, errors.Wrapf(err, "failed to marshal p2p data, project_id %v", t.ProjectID)
	}

	if err = networking.Publish(topic(t.ProjectID), data); err != nil {
		return 0, errors.Wrapf(err, "failed to publish data, project_id %v", t.ProjectID)
	}

	slog.Debug("dispatched a task", "project_id", t.ProjectID, "task_id", t.ID)
	return t.ID + 1, nil
}
