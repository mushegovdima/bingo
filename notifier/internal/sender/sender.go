package sender

import (
	"context"
	"errors"
	"fmt"
	"time"

	"notifier/internal/config"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// ErrPermanent is returned when the Telegram API rejects a message in a way that
// cannot be fixed by retrying (e.g. bot blocked, chat not found, user deactivated).
// Workers should mark such notifications as processed instead of retrying.
var ErrPermanent = errors.New("permanent telegram error")

// Sender is the interface that worker uses to deliver notifications.
type Sender interface {
	Send(ctx context.Context, chatID int64, text string) error
}

// NewSender returns the default Sender backed by Telegram Bot API.
func NewSender(cfg *config.Config) (Sender, error) {
	return newTelegramSender(cfg.TGBotToken)
}

type telegramSender struct {
	bot *tgbotapi.BotAPI
}

func newTelegramSender(token string) (*telegramSender, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("sender: init telegram: %w", err)
	}
	return &telegramSender{bot: bot}, nil
}

func (t *telegramSender) Send(ctx context.Context, chatID int64, text string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeHTML
	_, err := t.bot.Send(msg)
	if err == nil {
		return nil
	}

	var tgErr *tgbotapi.Error
	if errors.As(err, &tgErr) && tgErr.Code == 429 {
		select {
		case <-time.After(time.Duration(tgErr.RetryAfter) * time.Second):
		case <-ctx.Done():
			return ctx.Err()
		}
		if _, err = t.bot.Send(msg); err != nil {
			return fmt.Errorf("sender.Send retry: %w", err)
		}
		return nil
	}

	// 403 Forbidden (bot blocked / user deactivated) and 400 Bad Request
	// (chat not found, invalid chat ID) are permanent — retrying won't help.
	if errors.As(err, &tgErr) && (tgErr.Code == 403 || tgErr.Code == 400) {
		return fmt.Errorf("sender.Send: %w: telegram %d: %s", ErrPermanent, tgErr.Code, tgErr.Message)
	}

	return fmt.Errorf("sender.Send: %w", err)
}
