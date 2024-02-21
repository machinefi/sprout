package persistence

import (
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"

	"github.com/machinefi/sprout/project"
	"github.com/machinefi/sprout/types"
)

type message struct {
	gorm.Model
	MessageID      string `gorm:"index:message_id,not null"`
	ClientDID      string `gorm:"column:client_did;index:message_fetch,not null,default:''"`
	ProjectID      uint64 `gorm:"index:message_fetch,not null"`
	ProjectVersion string `gorm:"index:message_fetch,not null,default:'0.0'"`
	Data           string `gorm:"size:4096"`
	TaskID         string `gorm:"index:task_id,not null,default:''"`
}

type task struct {
	gorm.Model
	TaskID    string          `gorm:"index:task_id,not null"`
	MessageID string          `gorm:"index:message_id,not null"`
	State     types.TaskState `gorm:"index:task_fetch,not null"`
}

type taskStateLog struct {
	gorm.Model
	TaskID  string          `gorm:"index:task_id,not null"`
	State   types.TaskState `gorm:"not null"`
	Comment string
}

type Postgres struct {
	db *gorm.DB
}

func (p *Postgres) Save(msg *types.Message, config *project.Config) error {
	m := message{
		MessageID:      msg.ID,
		ClientDID:      msg.ClientDID,
		ProjectID:      msg.ProjectID,
		ProjectVersion: msg.ProjectVersion,
		Data:           msg.Data,
	}
	tid := uuid.NewString()
	ts := []task{{
		TaskID:    tid,
		MessageID: msg.ID,
		State:     types.TaskStatePacked,
	}}

	l := taskStateLog{
		TaskID: tid,
		State:  types.TaskStatePacked,
	}

	return p.db.Transaction(func(tx *gorm.DB) error {
		if a := config.Aggregation.Amount; a > 1 {
			ms := []*message{}
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Order("created_at").
				Where(
					"project_id = ? AND project_version = ? AND client_did = ? AND task_id = ?",
					msg.ProjectID, msg.ProjectVersion, msg.ClientDID, "",
				).Limit(int(a - 1)).Find(&ms).Error; err != nil {
				return errors.Wrap(err, "fetch message failed")
			}
			if len(ms) < int(a-1) {
				if err := tx.Create(&m).Error; err != nil {
					return errors.Wrap(err, "create message failed")
				}
				return nil
			}
			mids := []string{}
			for _, m := range ms {
				mids = append(mids, m.MessageID)
				ts = append(ts, task{
					TaskID:    tid,
					MessageID: m.MessageID,
					State:     types.TaskStatePacked,
				})
			}
			if err := tx.Model(message{}).Where("message_id IN ?", mids).Update("task_id", tid).Error; err != nil {
				return errors.Wrap(err, "update message taskID failed")
			}
		}

		m.TaskID = tid
		if err := tx.Create(&m).Error; err != nil {
			return errors.Wrap(err, "create message failed")
		}
		if err := tx.Create(&ts).Error; err != nil {
			return errors.Wrap(err, "create task failed")
		}
		if err := tx.Create(&l).Error; err != nil {
			return errors.Wrap(err, "create task state log failed")
		}
		return nil
	})
}

func (p *Postgres) Fetch() (*types.Task, error) {
	t := task{}
	if err := p.db.Where("state = ?", types.TaskStatePacked).First(&t).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errors.Wrap(err, "query task failed")
	}
	return p.FetchByID(t.TaskID)
}

func (p *Postgres) FetchByID(taskID string) (*types.Task, error) {
	ts := []*task{}
	if err := p.db.Where("task_id = ?", taskID).Find(&ts).Error; err != nil {
		return nil, errors.Wrap(err, "query task failed")
	}
	if len(ts) == 0 {
		return nil, nil
	}
	mids := []string{}
	for _, t := range ts {
		mids = append(mids, t.MessageID)
	}

	ms := []*message{}
	if err := p.db.Where("message_id IN ?", mids).Find(&ms).Error; err != nil {
		return nil, errors.Wrapf(err, "query message failed, taskID %s", taskID)
	}
	if len(ms) == 0 {
		return nil, errors.Errorf("missing message, taskID %s", taskID)
	}

	sort.Slice(ms, func(i, j int) bool {
		return ms[i].ID <= ms[j].ID
	})

	tms := []*types.Message{}
	for _, m := range ms {
		tms = append(tms, &types.Message{
			ID:             m.MessageID,
			ClientDID:      m.ClientDID,
			ProjectID:      m.ProjectID,
			ProjectVersion: m.ProjectVersion,
			Data:           m.Data,
		})
	}
	return &types.Task{
		ID:       taskID,
		Messages: tms,
	}, nil
}

func (p *Postgres) FetchMessage(messageID string) ([]*types.MessageWithTime, error) {
	ms := []*message{}
	if err := p.db.Where("message_id = ?", messageID).Find(&ms).Error; err != nil {
		return nil, errors.Wrapf(err, "query message by messageID failed, messageID %s", messageID)
	}

	tms := []*types.MessageWithTime{}
	for _, m := range ms {
		tms = append(tms, &types.MessageWithTime{
			Message: types.Message{
				ID:             m.MessageID,
				ClientDID:      m.ClientDID,
				ProjectID:      m.ProjectID,
				ProjectVersion: m.ProjectVersion,
				Data:           m.Data,
			},
			CreatedAt: m.CreatedAt,
		})
	}
	return tms, nil
}

func (p *Postgres) FetchStateLog(messageID string) ([]*types.TaskStateLog, error) {
	ts := []*task{}
	if err := p.db.Where("message_id = ?", messageID).Find(&ts).Error; err != nil {
		return nil, errors.Wrapf(err, "query task by messageID failed, messageID %s", messageID)
	}
	tids := []string{}
	for _, t := range ts {
		tids = append(tids, t.TaskID)
	}
	if len(tids) == 0 {
		return nil, nil
	}

	ls := []*taskStateLog{}
	if err := p.db.Order("created_at").Where("task_id IN ?", tids).Find(&ls).Error; err != nil {
		return nil, errors.Wrapf(err, "query task state log failed, taskIDs %v", tids)
	}

	tls := []*types.TaskStateLog{}
	for _, l := range ls {
		tls = append(tls, &types.TaskStateLog{
			TaskID:    l.TaskID,
			State:     l.State,
			Comment:   l.Comment,
			CreatedAt: l.CreatedAt,
		})
	}
	return tls, nil
}

func (p *Postgres) UpdateState(taskID string, state types.TaskState, comment string, createdAt time.Time) error {
	return p.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&task{}).Where("task_id = ?", taskID).Update("state", state).Error; err != nil {
			return errors.Wrapf(err, "update task state failed, task_id %s, target_state %s", taskID, state.String())
		}
		if err := tx.Create(&taskStateLog{
			TaskID:  taskID,
			State:   state,
			Comment: comment,
			Model: gorm.Model{
				CreatedAt: createdAt,
			},
		}).Error; err != nil {
			return errors.Wrap(err, "create task state log failed")
		}
		return nil
	})
}

func NewPostgres(pgEndpoint string) (*Postgres, error) {
	db, err := gorm.Open(postgres.Open(pgEndpoint), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, errors.Wrap(err, "connect postgres failed")
	}
	if err := db.AutoMigrate(&message{}, &task{}, &taskStateLog{}); err != nil {
		return nil, errors.Wrap(err, "migrate model failed")
	}
	return &Postgres{db}, nil
}
