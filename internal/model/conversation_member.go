package model

import "time"

type ConversationMember struct {
	ConversationID    int64     `json:"conversation_id"`
	UserID            int64     `json:"user_id"`
	JoinedAt          time.Time `json:"joined_at"`
	LastReadMessageID *int64    `json:"last_read_message_id,omitempty"`
}
