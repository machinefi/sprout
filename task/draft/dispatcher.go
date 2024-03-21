package draft

import (
	"encoding/json"
	"time"
)

func NewDispatcher(datasource Datasource, networking Networking, persistence Persistence, projects ProjectPool) *Dispatcher {
	d := &Dispatcher{}

	id, _ := d.datasource.Next()

	go d.dispatching(id)

	return d
}

type Dispatcher struct {
	datasource  Datasource
	networking  Networking
	persistence Persistence
	projects    ProjectPool
}

func (d *Dispatcher) dispatching(id uint64) {
	for {
		next, _ := d.dispatch(id)

		id = next
	}
}

func (d *Dispatcher) dispatch(id uint64) (next uint64, err error) {
	t, _ := d.datasource.Retrieve(id)

	data, _ := json.Marshal(t)

	_ = d.networking.Publish(topic(t.ProjectID), data)

	return t.ID + 1, nil
}

func (d *Dispatcher) Handle(input []byte) (output []byte) {
	// deserialize from input to TaskStateLog
	t := &TaskStateLog{} // input from networking
	r := &TaskStateLog{} // output to persistence

	if t.State != TaskStateProved {
		return
	}

	_ = d.persistence.Create(t)

	c, _ := d.projects.Get(t.Task.ProjectID, t.Task.ProjectVersion)

	o, _ := c.Output()

	result, err := o.Output(t.Task.ProjectID, t.Task.Data, t.Result)
	if err != nil {
		r.State = TaskStateFailed
		r.Comment = err.Error()
	} else {
		r.State = TaskStateOutputted
		r.Result = []byte(result)
	}

	r.CreatedAt = time.Now()
	r.Task = t.Task

	_ = d.persistence.Create(r)

	return
}
