package model

type ReadReceipt struct {
	ConversationID    int64 `json:"conversation_id"`
	UserID            int64 `json:"user_id"`
	LastReadMessageID int64 `json:"last_read_message_id"`
}
