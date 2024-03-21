package task

/*
import (
	"encoding/json"
	"reflect"
	"testing"

	. "github.com/agiledragon/gomonkey/v2"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/machinefi/sprout/output"
	"github.com/machinefi/sprout/p2p"
	"github.com/machinefi/sprout/project"
	"github.com/machinefi/sprout/testutil"
	testproject "github.com/machinefi/sprout/testutil/project"
	"github.com/machinefi/sprout/vm"
)

func TestNewProcessor(t *testing.T) {
	r := require.New(t)

	ps := &p2p.PubSubs{}

	t.Run("NewFailed", func(t *testing.T) {
		p := NewPatches()
		defer p.Reset()
		p = p.ApplyFuncReturn(p2p.NewPubSubs, nil, errors.New(t.Name()))
		_, err := NewProcessor(nil, nil, "", 0)
		r.ErrorContains(err, t.Name())
	})

	t.Run("AddProjectFailed", func(t *testing.T) {
		p := NewPatches()
		defer p.Reset()
		p = p.ApplyFuncReturn(p2p.NewPubSubs, ps, nil)
		p = testproject.ProjectManagerGetAllProjectID(p, append([]uint64{}, 1))
		p = p.ApplyMethodReturn(&p2p.PubSubs{}, "Add", errors.New(t.Name()))
		_, err := NewProcessor(nil, &project.Manager{}, "", 0)
		r.ErrorContains(err, t.Name())
	})

	t.Run("NewProcessorSuccess", func(t *testing.T) {
		p := NewPatches()
		defer p.Reset()
		p = p.ApplyFuncReturn(p2p.NewPubSubs, ps, nil)
		p = testproject.ProjectManagerGetAllProjectID(p, append([]uint64{}, 1))
		p = p.ApplyMethodReturn(&p2p.PubSubs{}, "Add", nil)

		pm := &project.Manager{}
		ch := make(chan uint64, 1)
		p = p.ApplyMethodReturn(&project.Manager{}, "GetNotify", ch)

		_, err := NewProcessor(nil, pm, "", 0)
		r.NoError(err)
	})
}

func TestProcessor_ReportFail(t *testing.T) {
	processor := &Processor{}

	t.Run("MarshalFailed", func(t *testing.T) {
		p := NewPatches()
		defer p.Reset()
		p = testutil.JsonMarshal(p, []byte("any"), errors.New(t.Name()))
		processor.reportFail(&Task{}, errors.New(t.Name()), nil)
	})

	t.Run("PublishFailed", func(t *testing.T) {
		p := NewPatches()
		defer p.Reset()
		p = testutil.JsonMarshal(p, []byte("any"), nil)

		p = testutil.TopicPublish(p, errors.New(t.Name()))
		processor.reportFail(&Task{}, errors.New(t.Name()), nil)
	})
}

func TestProcessor_ReportSuccess(t *testing.T) {
	processor := &Processor{}

	t.Run("MarshalFailed", func(t *testing.T) {
		p := NewPatches()
		defer p.Reset()
		p = testutil.JsonMarshal(p, []byte("any"), errors.New(t.Name()))
		processor.reportSuccess(&Task{}, TaskStatePacked, nil, nil)
	})

	t.Run("PublishFailed", func(t *testing.T) {
		p := NewPatches()
		defer p.Reset()
		p = testutil.JsonMarshal(p, []byte("any"), nil)

		p = testutil.TopicPublish(p, errors.New(t.Name()))
		processor.reportSuccess(&Task{}, TaskStatePacked, nil, nil)
	})

}

func TestProcessor_HandleP2PData(t *testing.T) {
	r := require.New(t)

	processor := &Processor{
		executor:       &vm.Handler{},
		projectManager: &project.Manager{},
		ps:             nil,
	}

	t.Run("TaskNil", func(t *testing.T) {
		data, err := json.Marshal(&p2pData{
			Task:         nil,
			TaskStateLog: nil,
		})
		r.NoError(err)

		processor.handleP2PData(data, nil)
	})

	data, err := json.Marshal(&p2pData{
		Task: &Task{
			ID:             1,
			ProjectID:      uint64(0x1),
			ProjectVersion: "0.1",
			Data:           [][]byte{[]byte("data")},
		},
		TaskStateLog: nil,
	})
	r.NoError(err)

	t.Run("GetProjectFailed", func(t *testing.T) {
		p := NewPatches()
		defer p.Reset()

		p = processorReportSuccess(p)
		p = testproject.ProjectManagerGet(p, nil, errors.New(t.Name()))
		p = processorReportFail(p)
		processor.handleP2PData(data, nil)
	})

	conf := &project.Config{
		Code:         "code",
		CodeExpParam: "codeExpParam",
		VMType:       "vmType",
		Output:       output.Config{},
		Aggregation:  project.AggregationConfig{},
		Version:      "",
	}

	t.Run("ProofFailed", func(t *testing.T) {
		p := NewPatches()
		defer p.Reset()
		p = testproject.ProjectManagerGet(p, conf, nil)

		p = processorReportSuccess(p)
		p = vmHandlerHandle(p, nil, errors.New(t.Name()))
		p = processorReportFail(p)
		processor.handleP2PData(data, nil)
	})

	t.Run("HandleSuccess", func(t *testing.T) {
		p := NewPatches()
		defer p.Reset()
		p = testproject.ProjectManagerGet(p, conf, nil)
		p = vmHandlerHandle(p, []byte("res"), nil)

		p = processorReportSuccess(p)
		processor.handleP2PData(data, nil)
	})

}

func processorReportSuccess(p *Patches) *Patches {
	var pro *Processor
	return p.ApplyPrivateMethod(pro, "reportSuccess", func(taskID string, state TaskState, comment string, topic *pubsub.Topic) {})
}

func processorReportFail(p *Patches) *Patches {
	var pro *Processor
	return p.ApplyPrivateMethod(pro, "reportFail", func(taskID string, err error, topic *pubsub.Topic) {})
}

func vmHandlerHandle(p *Patches, res []byte, err error) *Patches {
	var handler *vm.Handler
	return p.ApplyMethodFunc(
		reflect.TypeOf(handler),
		"Handle",
		func(projectID uint64, vmtype vm.Type, code string, expParam string, data [][]byte) ([]byte, error) {
			return res, err
		},
	)
}
*/
