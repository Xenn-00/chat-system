package chat_handler

import (
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/xenn00/chat-system/internal/dtos/chat_dto"
	"github.com/xenn00/chat-system/internal/queue"
	"github.com/xenn00/chat-system/internal/utils/types"
)

func (h *ChatHandler) broadcastPrivateMessage(resp *chat_dto.SendPrivateMessageResponse) error {
	jobPayload := &types.BroadcastMessagePayload{
		MessageID:  resp.MessageID,
		RoomID:     resp.RoomID,
		SenderID:   resp.SenderID,
		ReceiverID: resp.ReceiverID,
		Content:    resp.Content,

		CreatedAt: resp.CreatedAt,
	}

	job := queue.Job{
		ID:        uuid.New().String(),
		Type:      "broadcast_private_message",
		Payload:   queue.MustMarshal(jobPayload),
		Priority:  2,
		Retry:     0,
		MaxRetry:  3,
		CreatedAt: time.Now().Unix(),
		ExpireAt:  time.Now().Add(1 * time.Minute).Unix(),
	}

	if err := h.Producer.Enqueue(h.State.Ctx, job); err != nil {
		log.Error().Err(err).Msg("Failed to enqueue job")
		return err
	}

	log.Info().Str("job_id", job.ID).Str("message_id", resp.MessageID).Msg("Broadcast job enqueued successfully")
	return nil
}

func (h *ChatHandler) broadcastPrivateMessageReply(resp *chat_dto.ReplyPrivateMessageResponse) error {
	jobPayload := &types.BroadcastMessagePayload{
		MessageID:  resp.MessageID,
		RoomID:     resp.RoomID,
		SenderID:   resp.SenderID,
		ReceiverID: resp.ReceiverID,
		Content:    resp.Content,
		IsRead:     &resp.IsRead,
		ReplyTo: &types.ReplyTo{
			MessageID: resp.ReplyTo.RepliedMessageID,
			Content:   resp.ReplyTo.Content,
			SenderID:  resp.ReplyTo.SenderID,
		},
		CreatedAt: resp.CreatedAt,
	}

	job := queue.Job{
		ID:        uuid.New().String(),
		Type:      "broadcast_private_message_reply",
		Payload:   queue.MustMarshal(jobPayload),
		Priority:  1,
		Retry:     0,
		MaxRetry:  3,
		CreatedAt: time.Now().Unix(),
		ExpireAt:  time.Now().Add(1 * time.Minute).Unix(),
	}

	if err := h.Producer.Enqueue(h.State.Ctx, job); err != nil {
		log.Error().Err(err).Msg("Failed to enqueue job")
		return err
	}

	log.Info().Str("job_id", job.ID).Str("message_id", resp.MessageID).Msg("Broadcast job enqueued successfully")

	return nil
}

func (h *ChatHandler) broadcastPrivateMessageUpdated(resp *chat_dto.UpdatePrivateMessageResponse) {
	jobPayload := &types.BroadcastMessagePayload{
		MessageID:  resp.MessageID,
		RoomID:     resp.RoomID,
		SenderID:   resp.SenderID,
		ReceiverID: resp.ReceiverID,
		Content:    resp.Content,
		IsRead:     &resp.IsRead,
		IsEdited:   &resp.IsEdited,
		UpdatedAt:  &resp.UpdatedAt,
	}

	job := queue.Job{
		ID:        uuid.New().String(),
		Type:      "broadcast_private_message_updated",
		Payload:   queue.MustMarshal(jobPayload),
		Priority:  2,
		Retry:     0,
		MaxRetry:  3,
		CreatedAt: time.Now().Unix(),
		ExpireAt:  time.Now().Add(1 * time.Minute).Unix(),
	}

	if err := h.Producer.Enqueue(h.State.Ctx, job); err != nil {
		log.Error().Err(err).Msg("Failed to enqueue job")
	}
}
