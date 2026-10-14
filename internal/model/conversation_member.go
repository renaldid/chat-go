package model

import "time"

const (
	ConversationRoleOwner  = "owner"
	ConversationRoleAdmin  = "admin"
	ConversationRoleMember = "member"
)

type ConversationMember struct {
	ConversationID    int64     `json:"conversation_id"`
	UserID            int64     `json:"user_id"`
	Role              string    `json:"role"`
	JoinedAt          time.Time `json:"joined_at"`
	LastReadMessageID *int64    `json:"last_read_message_id,omitempty"`
}

type ConversationMemberView struct {
	ConversationID int64     `json:"conversation_id"`
	UserID         int64     `json:"user_id"`
	Username       string    `json:"username"`
	DisplayName    string    `json:"display_name"`
	Role           string    `json:"role"`
	JoinedAt       time.Time `json:"joined_at"`
}
