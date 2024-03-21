package task

import (
	"log/slog"
	"time"
)

func NewProcessor(executor ProofExecutor, projects ProjectPool) (*Processor, error) {
	p := &Processor{
		executor: executor,
		projects: projects,
	}

	return p, nil
}

type Processor struct {
	executor ProofExecutor
	projects ProjectPool
}

func (p *Processor) Handle(input []byte) (outputs [][]byte) {
	d := p2pData{}
	if err := d.Unmarshal(input); err != nil {
		slog.Error("failed to unmarshal p2p data", "error", err)
		return
	}
	if d.Task == nil {
		return
	}

	t := d.Task
	r := []*TaskStateLog{
		{Task: *d.Task, State: TaskStateDispatched},
		{Task: *d.Task},
	}

	defer func() {
		for _, v := range r {
			v.CreatedAt = time.Now()
			content, err := (&p2pData{TaskStateLog: v}).Marshal()
			if err != nil {
				continue
			}
			outputs = append(outputs, content)
		}
	}()

	slog.Debug("get a new task", "task_id", t.ID)

	config, err := p.projects.Get(t.ProjectID, t.ProjectVersion)
	if err != nil {
		slog.Error("failed to get project", "error", err, "project_id", t.ProjectID, "project_version", t.ProjectVersion)
		r[1].State = TaskStateFailed
		r[1].Comment = err.Error()
		return
	}

	res, err := p.executor.Handle(t.ProjectID, config.VMType, config.Code, config.CodeExpParam, t.Data)
	if err != nil {
		slog.Error("failed to generate proof", "error", err)
		r[1].State = TaskStateFailed
		r[1].Comment = err.Error()
		return
	}
	r[1].State = TaskStateProved
	r[1].Result = res
	return
}
