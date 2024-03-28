package datasource

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"

	taskpkg "github.com/machinefi/sprout/task"
)

type tablelandMessage struct {
	ID             uint
	MessageID      string
	ClientDID      string
	ProjectID      uint64
	ProjectVersion string
	Data           string // encode by hex
	InternalTaskID string
}

type tablelandTask struct {
	ID             uint
	InternalTaskID string
	MessageIDs     string
}

type tableland struct {
	gateway string
}

// message_11155111_1357 task_11155111_1358
func (t *tableland) Retrieve(nextTaskID uint64) (*taskpkg.Task, error) {
	taskSql := fmt.Sprintf("select * from task_11155111_1358 where id >= %d limit 1", nextTaskID)
	resp, err := http.Get(fmt.Sprintf("%s%s", t.gateway, taskSql))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query task, next_task_id %v", nextTaskID)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to query task from tableland.")
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()

	// TODO del
	fmt.Println(string(body))

	var tasks []tablelandTask
	if err := json.Unmarshal(body, &tasks); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal tasks, response is %v", string(body))
	}

	messageSql := fmt.Sprintf("select * from message_11155111_1357 where messageId = %s", tasks[0].MessageIDs)
	resp, err = http.Get(fmt.Sprintf("%s%s", t.gateway, messageSql))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query message, messageId %v", tasks[0].MessageIDs)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to query message from tableland.")
	}

	body, err = io.ReadAll(resp.Body)
	resp.Body.Close()

	// TODO del
	fmt.Println(string(body))

	var message []tablelandMessage
	if err := json.Unmarshal(body, &message); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal message, response is %v", string(body))
	}

	ds := [][]byte{}
	data, err := hex.DecodeString(message[0].Data[2:len(message[0].Data)])
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode message data, data is %v", message[0].Data)
	}
	ds = append(ds, data)

	return &taskpkg.Task{
		ID:             uint64(tasks[0].ID),
		ProjectID:      message[0].ProjectID,
		ProjectVersion: message[0].ProjectVersion,
		Data:           ds,
	}, nil
}

func NewTableland(dsn string) (Datasource, error) {
	// dsn https://testnets.tableland.network/api/v1/query?statement=
	return &tableland{dsn}, nil
}
