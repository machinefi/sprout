package task

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/facebookgo/clock"
	pubsub "github.com/libp2p/go-libp2p-pubsub"

	"github.com/machinefi/sprout/p2p"
	"github.com/machinefi/sprout/persistence"
	"github.com/machinefi/sprout/project"
	"github.com/machinefi/sprout/types"
)

type Dispatcher struct {
	pubSubs                   *p2p.PubSubs
	pg                        *persistence.Postgres
	projectManager            *project.Manager
	operatorPrivateKeyECDSA   string
	operatorPrivateKeyED25519 string
	ticker                    *clock.Ticker
}

func NewDispatcher(ticker *clock.Ticker, pg *persistence.Postgres, projectManager *project.Manager, bootNodeMultiaddr, operatorPrivateKey, operatorPrivateKeyED25519 string, iotexChainID int) (*Dispatcher, error) {
	d := &Dispatcher{
		ticker:                    ticker,
		pg:                        pg,
		projectManager:            projectManager,
		operatorPrivateKeyECDSA:   operatorPrivateKey,
		operatorPrivateKeyED25519: operatorPrivateKeyED25519,
	}
	ps, err := p2p.NewPubSubs(d.handleP2PData, bootNodeMultiaddr, iotexChainID)
	if err != nil {
		return nil, err
	}
	d.pubSubs = ps

	return d, nil
}

// Dispatch dispatches tasks
func (d *Dispatcher) Dispatch() {
	defer d.ticker.Stop()

	for range d.ticker.C {
		t, err := d.pg.Fetch()
		if err != nil {
			slog.Error("get task failed", "error", err)
			continue
		}
		if t == nil {
			continue
		}

		// TODO: check len of messages
		projectID := t.Messages[0].ProjectID
		if err := d.pubSubs.Add(projectID); err != nil {
			slog.Error("add project pubsub failed", "error", err, "projectID", projectID)
			continue
		}

		slog.Debug("dispatch project task", "projectID", projectID, "taskID", t.ID)
		if err := d.pubSubs.Publish(projectID, &p2p.Data{
			Task: t,
		}); err != nil {
			slog.Error("publish data failed", "error", err, "projectID", projectID)
		}
	}
}

func (d *Dispatcher) handleP2PData(data *p2p.Data, topic *pubsub.Topic) {
	if data.TaskStateLog == nil {
		return
	}
	l := data.TaskStateLog
	if err := d.pg.UpdateState(l.TaskID, l.State, l.Comment, l.CreatedAt); err != nil {
		slog.Error("update task state failed", "error", err, "taskID", l.TaskID)
		return
	}
	if l.State != types.TaskStateProved {
		return
	}

	task, err := d.pg.FetchByID(l.TaskID)
	if err != nil {
		slog.Error("fetch task failed", "error", err, "taskID", l.TaskID)
		return
	}
	pid := task.Messages[0].ProjectID
	pver := task.Messages[0].ProjectVersion
	config, err := d.projectManager.Get(pid, pver)
	if err != nil {
		slog.Error("get project failed", "error", err, "projectID", pid, "projectVersion", pver)
		return
	}

	output, err := config.GetOutput(d.operatorPrivateKeyECDSA, d.operatorPrivateKeyED25519)
	if err != nil {
		slog.Error("init output failed", "error", err)
		if err := d.pg.UpdateState(l.TaskID, types.TaskStateFailed, err.Error(), time.Now()); err != nil {
			slog.Error("update task state to statefailed failed", "error", err, "taskID", l.TaskID)
		}
		return
	}

	slog.Debug("output proof", "outputter", fmt.Sprintf("%T", output))

	outRes, err := output.Output(task, []byte(l.Comment))
	if err != nil {
		slog.Error("output failed", "error", err)
		if err := d.pg.UpdateState(l.TaskID, types.TaskStateFailed, err.Error(), time.Now()); err != nil {
			slog.Error("update task state to statefailed failed", "error", err, "taskID", l.TaskID)
		}
		return
	}

	if err := d.pg.UpdateState(l.TaskID, types.TaskStateOutputted, fmt.Sprintf("output result: %s", outRes), time.Now()); err != nil {
		slog.Error("update task state to outputted failed", "error", err, "taskID", l.TaskID)
	}
}
