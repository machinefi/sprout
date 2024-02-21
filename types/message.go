package types

import "time"

type Message struct {
	ID             string `json:"id"`
	ClientDID      string `json:"clientDID"`
	ProjectID      uint64 `json:"projectID"`
	ProjectVersion string `json:"projectVersion"`
	Data           string `json:"data"`
}

func (m *Message) GetData() *MessageData {
	return &MessageData{
		Data: m.Data,
	}
}

type MessageData struct {
	Data string
}

func (m *MessageData) Serialize() ([]byte, error) {
	return []byte(m.Data), nil
}

type MessageWithTime struct {
	Message
	CreatedAt time.Time
}
