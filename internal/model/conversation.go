package model

import "time"

const (
	ConversationTypeDirect = "direct"
	ConversationTypeGroup  = "group"
)

type Conversation struct {
	ID        int64     `json:"id"`
	Type      string    `json:"type"`
	Title     string    `json:"title,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ConversationSummary struct {
	ID          int64     `json:"id"`
	Type        string    `json:"type"`
	Title       string    `json:"title,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	UnreadCount int64     `json:"unread_count"`
}
