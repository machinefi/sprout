package main

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

type message struct {
	gorm.Model
	MessageID      string `gorm:"index:message_id,not null"`
	ClientDID      string `gorm:"column:client_did;index:message_fetch,not null,default:''"`
	ProjectID      uint64 `gorm:"index:message_fetch,not null"`
	ProjectVersion string `gorm:"index:message_fetch,not null,default:'0.0'"`
	Data           []byte `gorm:"size:4096"`
	InternalTaskID string `gorm:"index:internal_task_id,not null,default:''"`
}

type task struct {
	gorm.Model
	InternalTaskID string         `gorm:"index:internal_task_id,not null"`
	MessageIDs     datatypes.JSON `gorm:"not null"`
}

type persistence struct {
	db *gorm.DB
}

func (p *persistence) createMessageTx(tx *gorm.DB, m *message) error {
	if err := tx.Create(m).Error; err != nil {
		return errors.Wrap(err, "failed to create message")
	}
	return nil
}

func (p *persistence) aggregateTaskTx(tx *gorm.DB, amount int, m *message) error {
	messages := make([]*message, 0)
	if amount == 0 {
		amount = 1
	}

	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Order("created_at").
		Where(
			"project_id = ? AND project_version = ? AND client_did = ? AND internal_task_id = ?",
			m.ProjectID, m.ProjectVersion, m.ClientDID, "",
		).Limit(amount).Find(&messages).Error; err != nil {
		return errors.Wrap(err, "failed to fetch unpacked messages")
	}

	// no enough message for pack task
	if len(messages) < amount {
		return nil
	}

	taskID := uuid.NewString()
	messageIDs := make([]string, 0, amount)
	for _, v := range messages {
		messageIDs = append(messageIDs, v.MessageID)
	}
	if err := tx.Model(&message{}).Where("message_id IN ?", messageIDs).Update("internal_task_id", taskID).Error; err != nil {
		return errors.Wrap(err, "failed to update message internal task id")
	}
	messageIDsJson, err := json.Marshal(messageIDs)
	if err != nil {
		return errors.Wrap(err, "failed to marshal message id array")
	}

	if err := tx.Create(&task{
		InternalTaskID: taskID,
		MessageIDs:     messageIDsJson,
	}).Error; err != nil {
		return errors.Wrap(err, "failed to create task")
	}
	return nil
}

func (p *persistence) save(msg *message, aggregationAmount uint) error {
	return p.db.Transaction(func(tx *gorm.DB) error {
		if err := p.createMessageTx(tx, msg); err != nil {
			return err
		}
		if err := p.aggregateTaskTx(tx, int(aggregationAmount), msg); err != nil {
			return err
		}
		return nil
	})
}

func (p *persistence) fetchMessage(messageID string) ([]*message, error) {
	ms := []*message{}
	if err := p.db.Where("message_id = ?", messageID).Find(&ms).Error; err != nil {
		return nil, errors.Wrapf(err, "query message by messageID failed, messageID %s", messageID)
	}

	return ms, nil
}

func (p *persistence) fetchTask(internalTaskID string) ([]*task, error) {
	ts := []*task{}
	if err := p.db.Where("internal_task_id = ?", internalTaskID).Find(&ts).Error; err != nil {
		return nil, errors.Wrapf(err, "query task by internal task id failed, internal_task_id %s", internalTaskID)
	}

	return ts, nil
}

func newPersistence(pgEndpoint string) (*persistence, error) {
	db, err := gorm.Open(postgres.Open(pgEndpoint), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect postgres")
	}
	if err := db.AutoMigrate(&message{}, &task{}); err != nil {
		return nil, errors.Wrap(err, "failed to migrate model")
	}
	return &persistence{db}, nil
}
