package model

import "time"

type Message struct {
	ID               int64         `json:"id"`
	ConversationID   int64         `json:"conversation_id"`
	SenderID         int64         `json:"sender_id"`
	Content          string        `json:"content"`
	CreatedAt        time.Time     `json:"created_at"`
	EditedAt         *time.Time    `json:"edited_at,omitempty"`
	DeletedAt        *time.Time    `json:"deleted_at,omitempty"`
	ReplyToMessageID *int64        `json:"reply_to_message_id,omitempty"`
	ReplyTo          *MessageReply `json:"reply_to,omitempty"`
}

type MessageReply struct {
	ID                int64      `json:"id"`
	SenderID          int64      `json:"sender_id"`
	SenderUsername    string     `json:"sender_username"`
	SenderDisplayName string     `json:"sender_display_name"`
	Content           string     `json:"content"`
	CreatedAt         time.Time  `json:"created_at"`
	EditedAt          *time.Time `json:"edited_at,omitempty"`
	DeletedAt         *time.Time `json:"deleted_at,omitempty"`
}
