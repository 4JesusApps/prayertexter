package messaging

import "context"

type MessageSender interface {
	SendMessage(ctx context.Context, to string, body string) error
}
