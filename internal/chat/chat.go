package chat

import "context"

// Notifier sends messages to a chat space.
type Notifier interface {
	Send(ctx context.Context, message string) error
	SendThread(ctx context.Context, message string, threadKey string) error
}
