package grpchandler

import (
	"context"
	"log/slog"
	"time"

	"notifier/internal/db/repository"
	pb "notifier/internal/pb/notification/v1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type notificationRepo interface {
	Insert(ctx context.Context, n *repository.Notification) error
	BulkInsert(ctx context.Context, ns []repository.Notification) error
}

type Handler struct {
	pb.UnimplementedNotificationServiceServer
	repo   notificationRepo
	logger *slog.Logger
}

func New(repo notificationRepo, logger *slog.Logger) *Handler {
	return &Handler{repo: repo, logger: logger}
}

func (h *Handler) Send(ctx context.Context, req *pb.SendNotificationRequest) (*pb.SendNotificationResponse, error) {
	n := &repository.Notification{
		ID:         req.Id,
		Type:       req.Type,
		UserID:     req.UserId,
		TelegramID: req.TelegramId,
		Text:       req.Text,
		SendAt:     time.Now().UTC(),
	}

	if err := h.repo.Insert(ctx, n); err != nil {
		h.logger.ErrorContext(ctx, "failed to insert notification",
			slog.String("id", req.Id),
			slog.Any("error", err),
		)
		return nil, status.Error(codes.Internal, "failed to enqueue notification")
	}

	h.logger.InfoContext(ctx, "notification enqueued",
		slog.String("id", req.Id),
		slog.String("type", req.Type),
		slog.Int64("telegram_id", req.TelegramId),
	)

	return &pb.SendNotificationResponse{Id: req.Id}, nil
}

// SendBulk persists each notification as an independent outbox row.
func (h *Handler) SendBulk(ctx context.Context, req *pb.SendBulkNotificationRequest) (*pb.SendBulkNotificationResponse, error) {
	if len(req.Notifications) == 0 {
		return nil, status.Error(codes.InvalidArgument, "notifications must not be empty")
	}

	sendAt := time.Now().UTC()

	ns := make([]repository.Notification, len(req.Notifications))
	ids := make([]string, len(req.Notifications))
	for i, r := range req.Notifications {
		ns[i] = repository.Notification{
			ID:         r.Id,
			Type:       r.Type,
			UserID:     r.UserId,
			TelegramID: r.TelegramId,
			Text:       r.Text,
			SendAt:     sendAt,
		}
		ids[i] = r.Id
	}

	if err := h.repo.BulkInsert(ctx, ns); err != nil {
		h.logger.ErrorContext(ctx, "failed to bulk insert notifications",
			slog.Int("count", len(ns)),
			slog.Any("error", err),
		)
		return nil, status.Error(codes.Internal, "failed to enqueue notifications")
	}

	h.logger.InfoContext(ctx, "notifications bulk enqueued", slog.Int("count", len(ns)))
	return &pb.SendBulkNotificationResponse{Ids: ids}, nil
}
