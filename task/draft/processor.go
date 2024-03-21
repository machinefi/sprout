package draft

import "encoding/json"

func NewProcessor(executor ProofExecutor, projects ProjectPool, networking Networking) *Processor {
	p := &Processor{
		executor:   executor,
		projects:   projects,
		networking: networking,
	}
	return p
}

type Processor struct {
	executor   ProofExecutor
	projects   ProjectPool
	networking Networking
}

// handle impl interface p2pDataHandler
func (p *Processor) Handle(input []byte) (output []byte) {
	// need deserialize from data to *Task
	t := &Task{}
	r := &TaskStateLog{}

	defer func() {
		var err error
		output, err = json.Marshal(r)
		if err != nil {
			// log failed to serialize
		}
		return
	}()

	conf, err := p.projects.Get(t.ProjectID, t.ProjectVersion)
	if err != nil {
		r.State = TaskStateFailed
		r.Comment = err.Error()
		// log failed to get config from project pool
		return
	}

	proof, err := p.executor.Handle(t, conf)
	if err != nil {
		r.State = TaskStateFailed
		r.Comment = err.Error()
		// log failed to commit proof task
		return
	}

	r.State = TaskStateProved
	r.Result = proof

	return
}
