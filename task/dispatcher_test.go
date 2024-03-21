package task

/*
import (
	"encoding/json"
	"testing"
	"time"

	. "github.com/agiledragon/gomonkey/v2"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/machinefi/sprout/output"
	"github.com/machinefi/sprout/p2p"
	"github.com/machinefi/sprout/project"
	"github.com/machinefi/sprout/testutil/mock"
	testproject "github.com/machinefi/sprout/testutil/project"
)

type mockPersistence struct{}

func (m *mockPersistence) Create(tl *TaskStateLog) error {
	return nil
}

type mockDatasourceNil struct{}

func (m *mockDatasourceNil) Retrieve(nextTaskID uint64) (*Task, error) {
	return nil, nil
}

type mockDatasourceErr struct {
	err error
}

func (m *mockDatasourceErr) Retrieve(nextTaskID uint64) (*Task, error) {
	return nil, m.err
}

type mockDatasourceSuccess struct {
	task *Task
}

func (m *mockDatasourceSuccess) Retrieve(nextTaskID uint64) (*Task, error) {
	return m.task, nil
}

func TestNewDispatcher(t *testing.T) {
	r := require.New(t)

	t.Run("NewFailed", func(t *testing.T) {
		p := NewPatches()
		defer p.Reset()

		p = p.ApplyFuncReturn(p2p.NewPubSubs, nil, errors.New(t.Name()))
		_, err := NewDispatcher(nil, nil, nil, "", "", "", 0)
		r.ErrorContains(err, t.Name())
	})

	t.Run("New", func(t *testing.T) {
		p := NewPatches()
		defer p.Reset()
		p = p.ApplyFuncReturn(p2p.NewPubSubs, nil, nil)

		_, err := NewDispatcher(nil, nil, nil, "", "", "", 0)
		r.NoError(err)
	})
}

func TestDispatcher_HandleP2PData(t *testing.T) {
	r := require.New(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	op := mock.NewMockOutput(ctrl)

	d := &Dispatcher{
		pubSubs:                   nil,
		persistence:               &mockPersistence{},
		projects:                  &project.Manager{},
		operatorPrivateKeyECDSA:   "",
		operatorPrivateKeyED25519: "",
	}

	t.Run("TaskStateLogNil", func(t *testing.T) {
		data, err := json.Marshal(&p2pData{
			Task:         nil,
			TaskStateLog: nil,
		})
		r.NoError(err)
		d.handleP2PData(data, nil)
	})

	data, err := json.Marshal(&p2pData{
		Task: nil,
		TaskStateLog: &TaskStateLog{
			Task:      Task{ID: 1},
			State:     TaskStatePacked,
			Comment:   "Comment",
			CreatedAt: time.Now(),
		},
	})
	r.NoError(err)

	t.Run("FailedToCreateTaskStateLog", func(t *testing.T) {
		p := NewPatches()
		defer p.Reset()
		p = p.ApplyMethodReturn(&mockPersistence{}, "Create", errors.New(t.Name()))
		d.handleP2PData(data, nil)
	})

	t.Run("NotTaskStateProved", func(t *testing.T) {
		p := NewPatches()
		defer p.Reset()
		p = p.ApplyMethodReturn(&mockPersistence{}, "Create", nil)

		d.handleP2PData(data, nil)
	})

	data, err = json.Marshal(&p2pData{
		Task: nil,
		TaskStateLog: &TaskStateLog{
			Task:      Task{ID: 1},
			State:     TaskStateProved,
			Comment:   "Comment",
			CreatedAt: time.Now(),
		},
	})
	r.NoError(err)

	t.Run("FailedToGetProject", func(t *testing.T) {
		p := NewPatches()
		defer p.Reset()
		p = p.ApplyMethodReturn(&mockPersistence{}, "Create", nil)

		p = testproject.ProjectManagerGet(p, nil, errors.New(t.Name()))
		d.handleP2PData(data, nil)
	})

	t.Run("FailedToInitOutputAndFailedToCreateTaskStateLog", func(t *testing.T) {
		p := NewPatches()
		defer p.Reset()

		outputCell := []OutputCell{
			{Values: Params{nil}},
			{Values: Params{errors.New(t.Name())}},
		}
		p = p.ApplyMethodSeq(&mockPersistence{}, "Create", outputCell)
		p = testproject.ProjectManagerGet(p, &project.Config{}, nil)

		p = p.ApplyFuncReturn(output.New, nil, errors.New(t.Name()))
		d.handleP2PData(data, nil)
	})

	t.Run("FailedToInitOutput", func(t *testing.T) {
		p := NewPatches()
		defer p.Reset()

		outputCell := []OutputCell{
			{Values: Params{nil}},
			{Values: Params{nil}},
		}
		p = p.ApplyMethodSeq(&mockPersistence{}, "Create", outputCell)
		p = testproject.ProjectManagerGet(p, &project.Config{}, nil)

		p = p.ApplyFuncReturn(output.New, nil, errors.New(t.Name()))
		d.handleP2PData(data, nil)
	})

	t.Run("FailedToOutputAndFailedToCreateTaskStateLog", func(t *testing.T) {
		p := NewPatches()
		defer p.Reset()
		outputCell := []OutputCell{
			{Values: Params{nil}},
			{Values: Params{errors.New(t.Name())}},
		}
		p = p.ApplyMethodSeq(&mockPersistence{}, "Create", outputCell)
		p = testproject.ProjectManagerGet(p, &project.Config{}, nil)
		p = p.ApplyFuncReturn(output.New, op, nil)

		op.EXPECT().Output(gomock.Any(), gomock.Any(), gomock.Any()).Return("", errors.New(t.Name())).Times(1)
		d.handleP2PData(data, nil)
	})

	t.Run("FailedToOutput", func(t *testing.T) {
		p := NewPatches()
		defer p.Reset()
		outputCell := []OutputCell{
			{Values: Params{nil}},
			{Values: Params{nil}},
		}
		p = p.ApplyMethodSeq(&mockPersistence{}, "Create", outputCell)
		p = testproject.ProjectManagerGet(p, &project.Config{}, nil)
		p = p.ApplyFuncReturn(output.New, op, nil)

		op.EXPECT().Output(gomock.Any(), gomock.Any(), gomock.Any()).Return("", errors.New(t.Name())).Times(1)
		d.handleP2PData(data, nil)
	})

	t.Run("FailedToCreateOutputtedTaskState", func(t *testing.T) {
		p := NewPatches()
		defer p.Reset()
		outputCell := []OutputCell{
			{Values: Params{nil}},
			{Values: Params{errors.New(t.Name())}},
		}
		p = p.ApplyMethodSeq(&mockPersistence{}, "Create", outputCell)
		p = testproject.ProjectManagerGet(p, &project.Config{}, nil)
		p = p.ApplyFuncReturn(output.New, op, nil)
		op.EXPECT().Output(gomock.Any(), gomock.Any(), gomock.Any()).Return("", nil).Times(1)

		d.handleP2PData(data, nil)
	})

	t.Run("HandleP2PDataSuccess", func(t *testing.T) {
		p := NewPatches()
		defer p.Reset()
		outputCell := []OutputCell{
			{Values: Params{nil}},
			{Values: Params{nil}},
		}
		p = p.ApplyMethodSeq(&mockPersistence{}, "Create", outputCell)
		p = testproject.ProjectManagerGet(p, &project.Config{}, nil)
		p = p.ApplyFuncReturn(output.New, op, nil)
		op.EXPECT().Output(gomock.Any(), gomock.Any(), gomock.Any()).Return("", nil).Times(1)

		d.handleP2PData(data, nil)
	})
}

func TestDispatcher_DispatchTask(t *testing.T) {
	r := require.New(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := &Dispatcher{
		pubSubs:                   nil,
		persistence:               nil,
		projects:                  nil,
		operatorPrivateKeyECDSA:   "",
		operatorPrivateKeyED25519: "",
	}

	t.Run("FailedToRetrieve", func(t *testing.T) {
		d.datasource = &mockDatasourceErr{errors.New(t.Name())}
		_, err := d.dispatchTask(uint64(0x1))
		r.ErrorContains(err, t.Name())
	})

	t.Run("NilTask", func(t *testing.T) {
		d.datasource = &mockDatasourceNil{}
		taskId, err := d.dispatchTask(uint64(0x1))
		r.NoError(err)
		r.Equal(uint64(0x1), taskId)
	})

	t.Run("FailedToAdd", func(t *testing.T) {
		p := NewPatches()
		defer p.Reset()

		d.datasource = &mockDatasourceSuccess{&Task{ID: uint64(1)}}
		p = p.ApplyMethodReturn(&p2p.PubSubs{}, "Add", errors.New(t.Name()))
		_, err := d.dispatchTask(uint64(0x1))
		r.ErrorContains(err, t.Name())
	})

	t.Run("FailedToPublish", func(t *testing.T) {
		p := NewPatches()
		defer p.Reset()

		p = p.ApplyMethodReturn(&p2p.PubSubs{}, "Add", nil)
		p = p.ApplyMethodReturn(&p2p.PubSubs{}, "Publish", errors.New(t.Name()))
		_, err := d.dispatchTask(uint64(0x1))
		r.ErrorContains(err, t.Name())
	})

	t.Run("DispatchTaskSuccess", func(t *testing.T) {
		p := NewPatches()
		defer p.Reset()

		p = p.ApplyMethodReturn(&p2p.PubSubs{}, "Add", nil)
		p = p.ApplyMethodReturn(&p2p.PubSubs{}, "Publish", nil)
		taskId, err := d.dispatchTask(uint64(0x1))
		r.NoError(err)
		r.Equal(uint64(1)+1, taskId)
	})
}

func TestDispatcher_Dispatch(t *testing.T) {
	p := NewPatches()
	defer p.Reset()

	d := &Dispatcher{}

	t.Run("FailedToDispatchTask", func(t *testing.T) {
		ch := make(chan time.Time, 1)
		ticker := &time.Timer{C: ch}
		go func() { ch <- time.Now() }()
		p = p.ApplyFuncReturn(time.NewTimer, ticker)
		p = p.ApplyPrivateMethod(d, "dispatchTask", func(nextTaskID uint64) (uint64, error) {
			return 0, errors.New(t.Name())
		})
		go d.Dispatch(uint64(0x1))
		time.Sleep(1 * time.Second)
		close(ch)
	})

	t.Run("DispatchTaskSuccess", func(t *testing.T) {
		ch := make(chan time.Time, 1)
		ticker := &time.Timer{C: ch}
		go func() { ch <- time.Now() }()
		p = p.ApplyFuncReturn(time.NewTimer, ticker)
		p = p.ApplyPrivateMethod(d, "dispatchTask", func(nextTaskID uint64) (uint64, error) {
			return 0, nil
		})
		go d.Dispatch(uint64(0x1))
		time.Sleep(1 * time.Second)
		close(ch)
	})
}
*/
