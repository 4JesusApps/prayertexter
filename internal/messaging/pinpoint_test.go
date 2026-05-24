package messaging_test

import (
	"context"
	"strings"
	"testing"

	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/aws/aws-sdk-go-v2/service/pinpointsmsvoicev2"
	"github.com/aws/smithy-go"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	msgmocks "github.com/4JesusApps/prayertexter/internal/mocks/messaging"
)

type PinpointSenderSuite struct {
	suite.Suite
	client *msgmocks.MockPinpointClient
	sender *messaging.PinpointSender
	ctx    context.Context
}

func (s *PinpointSenderSuite) SetupTest() {
	s.client = msgmocks.NewMockPinpointClient(s.T())
	s.sender = messaging.NewPinpointSender(s.client, "test-pool", 60)
	s.ctx = context.Background()
}

func (s *PinpointSenderSuite) TestSendMessage_Success() {
	s.client.EXPECT().
		SendTextMessage(mock.Anything, mock.Anything).
		Return(&pinpointsmsvoicev2.SendTextMessageOutput{}, nil).
		Once()

	err := s.sender.SendMessage(s.ctx, "+11234567890", "hello")
	s.Require().NoError(err)
}

func (s *PinpointSenderSuite) TestSendMessage_ThrottleRetrySuccess() {
	throttleErr := &smithy.GenericAPIError{Code: "ThrottlingException", Message: "rate exceeded"}

	s.client.EXPECT().
		SendTextMessage(mock.Anything, mock.Anything).
		Return(nil, throttleErr).Once()
	s.client.EXPECT().
		SendTextMessage(mock.Anything, mock.Anything).
		Return(&pinpointsmsvoicev2.SendTextMessageOutput{}, nil).Once()

	err := s.sender.SendMessage(s.ctx, "+11234567890", "hello")
	s.Require().NoError(err)
}

func (s *PinpointSenderSuite) TestSendMessage_ThrottleExhaustRetries() {
	throttleErr := &smithy.GenericAPIError{Code: "ThrottlingException", Message: "rate exceeded"}

	s.client.EXPECT().
		SendTextMessage(mock.Anything, mock.Anything).
		Return(nil, throttleErr).Times(3)

	err := s.sender.SendMessage(s.ctx, "+11234567890", "hello")
	s.Require().Error(err)
}

func (s *PinpointSenderSuite) TestSendMessage_NonThrottleErrorNoRetry() {
	apiErr := &smithy.GenericAPIError{Code: "InternalServerError", Message: "boom"}

	s.client.EXPECT().
		SendTextMessage(mock.Anything, mock.Anything).
		Return(nil, apiErr).Once()

	err := s.sender.SendMessage(s.ctx, "+11234567890", "hello")
	s.Require().Error(err)
}

func (s *PinpointSenderSuite) TestSendMessage_AWSSAMLocalSkipsSend() {
	s.T().Setenv("AWS_SAM_LOCAL", "true")

	// No EXPECT calls configured; the mock will fail loudly if SendTextMessage is invoked.
	err := s.sender.SendMessage(s.ctx, "+11234567890", "hello")
	s.Require().NoError(err)
}

func (s *PinpointSenderSuite) TestSendMessage_BodyWrappingAndPhonePool() {
	s.client.EXPECT().
		SendTextMessage(mock.Anything, mock.MatchedBy(func(input *pinpointsmsvoicev2.SendTextMessageInput) bool {
			if input.MessageBody == nil || input.DestinationPhoneNumber == nil || input.OriginationIdentity == nil {
				return false
			}
			body := *input.MessageBody
			return strings.HasPrefix(body, messaging.MsgPre) &&
				strings.HasSuffix(body, messaging.MsgPost) &&
				strings.Contains(body, "hello world") &&
				*input.DestinationPhoneNumber == "+11234567890" &&
				*input.OriginationIdentity == "test-pool"
		})).
		Return(&pinpointsmsvoicev2.SendTextMessageOutput{}, nil).
		Once()

	err := s.sender.SendMessage(s.ctx, "+11234567890", "hello world")
	s.Require().NoError(err)
}

func TestPinpointSenderSuite(t *testing.T) {
	suite.Run(t, new(PinpointSenderSuite))
}
