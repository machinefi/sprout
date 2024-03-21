package task

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/machinefi/sprout/project"
	"github.com/machinefi/sprout/vm"
)

type Task struct {
	ID             uint64   `json:"id"`
	ProjectID      uint64   `json:"projectID"`
	ProjectVersion string   `json:"projectVersion"`
	Data           [][]byte `json:"data"`
}

type TaskState uint8

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

type TaskStateLog struct {
	Task      Task
	State     TaskState
	Comment   string
	Result    []byte
	CreatedAt time.Time
}

func (s TaskState) String() string {
	switch s {
	case TaskStatePacked:
		return "packed"
	case TaskStateDispatched:
		return "dispatched"
	case TaskStateProved:
		return "proved"
	case TaskStateOutputted:
		return "outputted"
	case TaskStateFailed:
		return "failed"
	default:
		return "invalid"
	}
}

func topic(projectID uint64) string {
	return "w3bstream-project-" + strconv.FormatUint(projectID, 10)
}

type p2pData struct {
	Task         *Task         `json:"task,omitempty"`
	TaskStateLog *TaskStateLog `json:"taskStateLog,omitempty"`
}

func (p *p2pData) Unmarshal(data []byte) error {
	return json.Unmarshal(data, p)
}

func (p *p2pData) Marshal() ([]byte, error) {
	return json.Marshal(p)
}

type Datasource interface {
	Retrieve(nextTaskID uint64) (*Task, error)
}

type Persistence interface {
	Create(tl *TaskStateLog) error
}

type Networking interface {
	Publish(topic string, data []byte) error
}

type ProjectPool interface {
	Get(projectID uint64, version string) (*project.Config, error)
}

type ProofExecutor interface {
	Handle(projectID uint64, vmtype vm.Type, code string, expParam string, data [][]byte) ([]byte, error)
}
