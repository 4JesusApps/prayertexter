package repository_test

import (
	"context"
	"testing"

	"github.com/4JesusApps/prayertexter/internal/domain"
	"github.com/4JesusApps/prayertexter/internal/repository"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	repomocks "github.com/4JesusApps/prayertexter/internal/mocks/repository"
)

type PhonesRepoSuite struct {
	suite.Suite
	client      *repomocks.MockDDBClient
	blocked     repository.BlockedPhonesRepository
	intercessor repository.IntercessorPhonesRepository
	ctx         context.Context
}

func (s *PhonesRepoSuite) SetupTest() {
	s.client = repomocks.NewMockDDBClient(s.T())
	s.blocked = repository.NewBlockedPhonesRepository(s.client, "General", 60)
	s.intercessor = repository.NewIntercessorPhonesRepository(s.client, "General", 60)
	s.ctx = context.Background()
}

func (s *PhonesRepoSuite) TestBlockedGet_NotFoundReturnsEmpty() {
	s.client.EXPECT().GetItem(mock.Anything, mock.Anything).
		Return(&dynamodb.GetItemOutput{}, nil)

	phones, err := s.blocked.Get(s.ctx)
	s.Require().NoError(err)
	s.NotNil(phones)
	s.Empty(phones.Phones)
}

func (s *PhonesRepoSuite) TestBlockedGet_ReturnsDomainWithoutKey() {
	row := map[string]types.AttributeValue{
		"Key": &types.AttributeValueMemberS{Value: "BlockedPhones"},
		"Phones": &types.AttributeValueMemberL{Value: []types.AttributeValue{
			&types.AttributeValueMemberS{Value: "+11111111111"},
			&types.AttributeValueMemberS{Value: "+12222222222"},
		}},
	}
	s.client.EXPECT().GetItem(mock.Anything, mock.MatchedBy(func(input *dynamodb.GetItemInput) bool {
		key, ok := input.Key["Key"].(*types.AttributeValueMemberS)
		return ok && key.Value == "BlockedPhones"
	})).Return(&dynamodb.GetItemOutput{Item: row}, nil)

	phones, err := s.blocked.Get(s.ctx)
	s.Require().NoError(err)
	s.Equal([]string{"+11111111111", "+12222222222"}, phones.Phones)
}

func (s *PhonesRepoSuite) TestBlockedSave_AttachesPartitionKey() {
	s.client.EXPECT().PutItem(mock.Anything, mock.MatchedBy(func(input *dynamodb.PutItemInput) bool {
		key, ok := input.Item["Key"].(*types.AttributeValueMemberS)
		return ok && key.Value == "BlockedPhones"
	})).Return(&dynamodb.PutItemOutput{}, nil)

	err := s.blocked.Save(s.ctx, &domain.BlockedPhones{Phones: []string{"+11111111111"}})
	s.Require().NoError(err)
}

func (s *PhonesRepoSuite) TestIntercessorGet_NotFoundReturnsEmpty() {
	s.client.EXPECT().GetItem(mock.Anything, mock.Anything).
		Return(&dynamodb.GetItemOutput{}, nil)

	phones, err := s.intercessor.Get(s.ctx)
	s.Require().NoError(err)
	s.NotNil(phones)
	s.Empty(phones.Phones)
}

func (s *PhonesRepoSuite) TestIntercessorGet_UsesIntercessorKey() {
	row := map[string]types.AttributeValue{
		"Key": &types.AttributeValueMemberS{Value: "IntercessorPhones"},
		"Phones": &types.AttributeValueMemberL{Value: []types.AttributeValue{
			&types.AttributeValueMemberS{Value: "+18888888888"},
		}},
	}
	s.client.EXPECT().GetItem(mock.Anything, mock.MatchedBy(func(input *dynamodb.GetItemInput) bool {
		key, ok := input.Key["Key"].(*types.AttributeValueMemberS)
		return ok && key.Value == "IntercessorPhones"
	})).Return(&dynamodb.GetItemOutput{Item: row}, nil)

	phones, err := s.intercessor.Get(s.ctx)
	s.Require().NoError(err)
	s.Equal([]string{"+18888888888"}, phones.Phones)
}

func (s *PhonesRepoSuite) TestIntercessorSave_AttachesPartitionKey() {
	s.client.EXPECT().PutItem(mock.Anything, mock.MatchedBy(func(input *dynamodb.PutItemInput) bool {
		key, ok := input.Item["Key"].(*types.AttributeValueMemberS)
		return ok && key.Value == "IntercessorPhones"
	})).Return(&dynamodb.PutItemOutput{}, nil)

	err := s.intercessor.Save(s.ctx, &domain.IntercessorPhones{Phones: []string{"+18888888888"}})
	s.Require().NoError(err)
}

func TestPhonesRepoSuite(t *testing.T) {
	suite.Run(t, new(PhonesRepoSuite))
}
