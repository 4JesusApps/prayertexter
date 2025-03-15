package mock

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/pinpointsmsvoicev2"
)

type TextSender struct {
	SendTextCalls   int
	SendTextInputs  []*pinpointsmsvoicev2.SendTextMessageInput
	SendTextResults []struct {
		Error error
	}
}

func (m *TextSender) SendTextMessage(_ context.Context, params *pinpointsmsvoicev2.SendTextMessageInput, _ ...func(
	*pinpointsmsvoicev2.Options)) (*pinpointsmsvoicev2.SendTextMessageOutput, error) {

	m.SendTextCalls++
	m.SendTextInputs = append(m.SendTextInputs, params)

	// Default result if no results are configured to avoid index out of bounds
	if len(m.SendTextResults) <= m.SendTextCalls-1 {
		return &pinpointsmsvoicev2.SendTextMessageOutput{}, nil
	}

	result := m.SendTextResults[m.SendTextCalls-1]

	return &pinpointsmsvoicev2.SendTextMessageOutput{}, result.Error
}
