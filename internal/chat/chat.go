package chat

import "context"

// Notifier sends messages to a chat space.
type Notifier interface {
	Send(ctx context.Context, message string) error
	SendThread(ctx context.Context, message string, threadKey string) error
	// SendAdmin sends a message to the admin/health channel. If no separate
	// admin channel is configured, implementations fall back to SendThread.
	SendAdmin(ctx context.Context, message string, threadKey string) error
}
