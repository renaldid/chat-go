package service

import (
	"context"
	"errors"
	"testing"
)

func TestReadReceiptServiceMarkRead(t *testing.T) {
	userRepo := newMockUserRepository()
	conversationRepo := newMockConversationRepository()
	memberRepo := newMockConversationMemberRepository()
	messageRepo := newMockMessageRepository()

	userService := NewUserService(userRepo)
	conversationService := NewConversationService(conversationRepo, memberRepo)

	user, err := userService.Create(context.Background(), "reader", "Reader", "password123")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	conversation, err := conversationService.Create(context.Background(), "Read Test", user.ID)
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	messageService := NewMessageService(messageRepo, conversationRepo, userRepo, memberRepo)

	message, err := messageService.Create(context.Background(), conversation.ID, user.ID, "hello")
	if err != nil {
		t.Fatalf("create message: %v", err)
	}

	svc := NewReadReceiptService(conversationRepo, memberRepo, messageRepo)

	receipt, err := svc.MarkRead(context.Background(), conversation.ID, user.ID, message.ID)
	if err != nil {
		t.Fatalf("mark read: %v", err)
	}

	if receipt.LastReadMessageID != message.ID {
		t.Fatalf("expected last read message id %d, got %d", message.ID, receipt.LastReadMessageID)
	}
}

func TestReadReceiptServiceMessageNotFound(t *testing.T) {
	conversationRepo := newMockConversationRepository()
	memberRepo := newMockConversationMemberRepository()
	messageRepo := newMockMessageRepository()
	conversationService := NewConversationService(conversationRepo, memberRepo)

	conversation, err := conversationService.Create(context.Background(), "Read Test", 1)
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	svc := NewReadReceiptService(conversationRepo, memberRepo, messageRepo)

	_, err = svc.MarkRead(context.Background(), conversation.ID, 1, 999)
	if !errors.Is(err, ErrMessageNotFound) {
		t.Fatalf("expected ErrMessageNotFound, got %v", err)
	}
}

func TestReadReceiptServiceMessageWrongConversation(t *testing.T) {
	userRepo := newMockUserRepository()
	conversationRepo := newMockConversationRepository()
	memberRepo := newMockConversationMemberRepository()
	messageRepo := newMockMessageRepository()

	userService := NewUserService(userRepo)
	conversationService := NewConversationService(conversationRepo, memberRepo)

	user, err := userService.Create(context.Background(), "reader", "Reader", "password123")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	conversation1, err := conversationService.Create(context.Background(), "Chat 1", user.ID)
	if err != nil {
		t.Fatalf("create conversation 1: %v", err)
	}

	conversation2, err := conversationService.Create(context.Background(), "Chat 2", user.ID)
	if err != nil {
		t.Fatalf("create conversation 2: %v", err)
	}

	messageService := NewMessageService(messageRepo, conversationRepo, userRepo, memberRepo)

	message, err := messageService.Create(context.Background(), conversation2.ID, user.ID, "hello")
	if err != nil {
		t.Fatalf("create message: %v", err)
	}

	svc := NewReadReceiptService(conversationRepo, memberRepo, messageRepo)

	_, err = svc.MarkRead(context.Background(), conversation1.ID, user.ID, message.ID)
	if !errors.Is(err, ErrMessageNotInConversation) {
		t.Fatalf("expected ErrMessageNotInConversation, got %v", err)
	}
}

func TestReadReceiptDoesNotMoveBackward(t *testing.T) {
	userRepo := newMockUserRepository()
	conversationRepo := newMockConversationRepository()
	memberRepo := newMockConversationMemberRepository()
	messageRepo := newMockMessageRepository()

	userService := NewUserService(userRepo)
	conversationService := NewConversationService(conversationRepo, memberRepo)

	user, _ := userService.Create(context.Background(), "reader", "Reader", "password123")
	conversation, _ := conversationService.Create(context.Background(), "Read Test", user.ID)

	messageService := NewMessageService(messageRepo, conversationRepo, userRepo, memberRepo)

	message1, _ := messageService.Create(context.Background(), conversation.ID, user.ID, "one")
	message2, _ := messageService.Create(context.Background(), conversation.ID, user.ID, "two")

	svc := NewReadReceiptService(conversationRepo, memberRepo, messageRepo)

	_, err := svc.MarkRead(context.Background(), conversation.ID, user.ID, message2.ID)
	if err != nil {
		t.Fatalf("mark second message read: %v", err)
	}

	receipt, err := svc.MarkRead(context.Background(), conversation.ID, user.ID, message1.ID)
	if err != nil {
		t.Fatalf("mark older message read: %v", err)
	}

	if receipt.LastReadMessageID != message2.ID {
		t.Fatalf("expected read pointer to remain %d, got %d", message2.ID, receipt.LastReadMessageID)
	}
}
