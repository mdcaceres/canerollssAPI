package output

import "context"

type MessageSender interface {
	SendTextMessage(ctx context.Context, to, message string) error
}
