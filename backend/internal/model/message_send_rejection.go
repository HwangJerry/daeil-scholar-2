// message_send_rejection.go — Typed reasons a POST /api/messages send is refused.
package model

// Stable error codes the mobile apps branch on for a refused message send.
const (
	MessageSendInvalid           = "MESSAGE_INVALID"
	MessageSendContentRejected   = "MESSAGE_CONTENT_REJECTED"
	MessageSendRecipientGone     = "RECIPIENT_UNAVAILABLE"
	MessageSendBlockedByMe       = "RECIPIENT_BLOCKED_BY_ME"
	MessageSendReceivingDisabled = "RECIPIENT_RECEIVING_DISABLED"
)

// MessageSendRejection is a client-facing refusal of a message send. Code is
// one of the MessageSend* constants; Message is the user-visible Korean copy.
type MessageSendRejection struct {
	Code    string
	Message string
}

func (e *MessageSendRejection) Error() string { return e.Code + ": " + e.Message }

func NewMessageInvalid(message string) *MessageSendRejection {
	return &MessageSendRejection{Code: MessageSendInvalid, Message: message}
}

func NewMessageContentRejected() *MessageSendRejection {
	return &MessageSendRejection{Code: MessageSendContentRejected, Message: "보낼 수 없는 내용이 포함되어 있어요."}
}

func NewMessageRecipientUnavailable() *MessageSendRejection {
	return &MessageSendRejection{Code: MessageSendRecipientGone, Message: "더 이상 쪽지를 보낼 수 없는 상대예요."}
}

func NewMessageRecipientBlockedByMe() *MessageSendRejection {
	return &MessageSendRejection{Code: MessageSendBlockedByMe, Message: "차단을 해제하면 메시지를 보낼 수 있습니다."}
}

func NewMessageReceivingDisabled() *MessageSendRejection {
	return &MessageSendRejection{Code: MessageSendReceivingDisabled, Message: "상대방이 쪽지 수신을 꺼두었어요."}
}
