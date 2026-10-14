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
	ID           int64                    `json:"id"`
	Type         string                   `json:"type"`
	Title        string                   `json:"title,omitempty"`
	DisplayTitle string                   `json:"display_title"`
	CreatedAt    time.Time                `json:"created_at"`
	UpdatedAt    time.Time                `json:"updated_at"`
	UnreadCount  int64                    `json:"unread_count"`
	MemberCount  int64                    `json:"member_count"`
	DirectUser   *ConversationUserSummary `json:"direct_user,omitempty"`
	LastMessage  *ConversationLastMessage `json:"last_message,omitempty"`
}

type ConversationUserSummary struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
}

type ConversationLastMessage struct {
	ID        int64                   `json:"id"`
	Sender    ConversationUserSummary `json:"sender"`
	Content   string                  `json:"content"`
	CreatedAt time.Time               `json:"created_at"`
}
