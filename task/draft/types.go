package draft

import (
	"github.com/machinefi/sprout/output"
	"github.com/machinefi/sprout/task"
	"github.com/machinefi/sprout/vm"
)

type (
	Task         = task.Task
	TaskStateLog = task.TaskStateLog
	TaskState    = task.TaskState
)

const (
	TaskStateInvalid TaskState = iota
	TaskStatePacked
	TaskStateDispatched
	_
	TaskStateProved
	_
	TaskStateOutputted
	TaskStateFailed
)

type Networking interface {
	// Publish data to topic
	Publish(topic string, data []byte) error
}

type ProofExecutor interface {
	// Handle execute/commit proof task
	Handle(task *Task, conf ProjectConfig) (result []byte, err error)
}

type P2PDataHandler interface {
	// Handle input data from networking, output as result to publish
	Handle(input []byte) (output []byte)
}

type ProjectPool interface {
	// GetConfig get project
	// consider combine GetConfig and GetOutput, project output should be initialized when project upserted.
	Get(id uint64, version string) (ProjectConfig, error)
	// ProjectIDs fetch project id list
	ProjectIDs() []uint64
}

type TopicEventSubscriber interface {
	// Subscribe returns event chan of pool modification
	Subscribe() <-chan *ProjectPoolMonitorEvent
}

type ProjectConfig interface {
	Type() vm.Type
	Code() string
	Param() string
	Output() (output.Output, error)
}

type ProjectPoolMonitorEvent struct {
	ProjectID uint64
	Action    int // added/removed/updated
}

type Datasource interface {
	Retrieve(id uint64) (*Task, error)
	Next() (uint64, error)
}

type Persistence interface {
	Create(tl *TaskStateLog) error
}

/*
                                                                                                                 gateway
                                                                                                                    |
                                                                                                                    V
                                                                                                                    da <----------
                                                                                                                    |            |
                                                                                                                    V            |
                   contract event                          topic event                           publish                         |
chain monitor -----------------------> project pool ---------------------------> networking  <-----------------  dispatcher      |
                                                                                     |                                           |
                                                                                     |          subscribe                        |
                                                                                     ------------------------->  processor    ----
*/
