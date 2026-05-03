package notifier

import (
	"context"
	"crypto/rand"
	"fmt"

	pb "go.mod/internal/pb/notification/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// MaxBulkRecipients is the upper bound on recipients per SendBulk call,
// matching the server-side limit and proto validator.
const MaxBulkRecipients = 500

// Sender is the interface consumed by services that enqueue notifications.
type Sender interface {
	Send(ctx context.Context, userID, telegramID int64, text string) error
	// SendBulk fans out personalised messages to up to MaxBulkRecipients
	// recipients in one unary RPC. Each BulkRecipient carries its own Text.
	SendBulk(ctx context.Context, notificationType string, recipients []BulkRecipient) error
}

// BulkRecipient is a single addressee in a bulk notification request.
type BulkRecipient struct {
	UserID     int64
	TelegramID int64
	Text       string // per-recipient rendered message body
}

// Client sends notifications via the notifier gRPC service.
type Client struct {
	grpc pb.NotificationServiceClient
}

// NewClient dials addr and returns a Client. The caller is responsible for
// closing the returned *grpc.ClientConn when done.
//
// Retry policy: up to 3 attempts on UNAVAILABLE and RESOURCE_EXHAUSTED with
// exponential backoff (0.1s–1s). This covers transient notifier restarts.
// Permanent errors (e.g. INVALID_ARGUMENT) are not retried.
func NewClient(addr string) (*Client, *grpc.ClientConn, error) {
	const serviceConfig = `{
		"methodConfig": [{
			"name": [{"service": "notification.v1.NotificationService"}],
			"retryPolicy": {
				"maxAttempts": 3,
				"initialBackoff": "0.1s",
				"maxBackoff": "1s",
				"backoffMultiplier": 2,
				"retryableStatusCodes": ["UNAVAILABLE", "RESOURCE_EXHAUSTED"]
			}
		}]
	}`
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(serviceConfig),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("notifier.NewClient: dial %s: %w", addr, err)
	}
	return &Client{grpc: pb.NewNotificationServiceClient(conn)}, conn, nil
}

func (c *Client) Send(ctx context.Context, userID, telegramID int64, text string) error {
	id, err := newUUID()
	if err != nil {
		return fmt.Errorf("notifier.Client.Send: generate id: %w", err)
	}
	_, err = c.grpc.Send(ctx, &pb.SendNotificationRequest{
		Id:         id,
		Type:       "season_available",
		UserId:     userID,
		TelegramId: telegramID,
		Text:       text,
	})
	if err != nil {
		return fmt.Errorf("notifier.Client.Send: %w", err)
	}
	return nil
}

func (c *Client) SendBulk(ctx context.Context, notificationType string, recipients []BulkRecipient) error {
	if len(recipients) == 0 {
		return nil
	}
	if len(recipients) > MaxBulkRecipients {
		return fmt.Errorf("notifier.Client.SendBulk: %d recipients exceeds max %d", len(recipients), MaxBulkRecipients)
	}
	notifications := make([]*pb.SendNotificationRequest, len(recipients))
	for i, r := range recipients {
		id, err := newUUID()
		if err != nil {
			return fmt.Errorf("notifier.Client.SendBulk: generate id: %w", err)
		}
		notifications[i] = &pb.SendNotificationRequest{
			Id:         id,
			Type:       notificationType,
			UserId:     r.UserID,
			TelegramId: r.TelegramID,
			Text:       r.Text,
		}
	}
	if _, err := c.grpc.SendBulk(ctx, &pb.SendBulkNotificationRequest{
		Notifications: notifications,
	}); err != nil {
		return fmt.Errorf("notifier.Client.SendBulk: %w", err)
	}
	return nil
}

// Noop is a no-op Sender used when the notifier service is not configured.
type Noop struct{}

func (Noop) Send(_ context.Context, _, _ int64, _ string) error { return nil }
func (Noop) SendBulk(_ context.Context, _ string, _ []BulkRecipient) error {
	return nil
}

func newUUID() (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	buf[6] = (buf[6] & 0x0f) | 0x40 // version 4
	buf[8] = (buf[8] & 0x3f) | 0x80 // variant
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		buf[0:4], buf[4:6], buf[6:8], buf[8:10], buf[10:16]), nil
}
