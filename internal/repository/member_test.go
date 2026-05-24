package repository_test

import (
	"context"
	"errors"
	"testing"

	"github.com/4JesusApps/prayertexter/internal/domain"
	"github.com/4JesusApps/prayertexter/internal/repository"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	repomocks "github.com/4JesusApps/prayertexter/internal/mocks/repository"
)

type MemberRepoSuite struct {
	suite.Suite
	client *repomocks.MockDDBClient
	repo   repository.MemberRepository
	ctx    context.Context
}

func (s *MemberRepoSuite) SetupTest() {
	s.client = repomocks.NewMockDDBClient(s.T())
	s.repo = repository.NewMemberRepository(s.client, "Member", 60)
	s.ctx = context.Background()
}

func (s *MemberRepoSuite) TestExists_Found() {
	av, _ := attributevalue.MarshalMap(&domain.Member{Phone: "+11234567890"})
	s.client.EXPECT().GetItem(mock.Anything, mock.Anything).
		Return(&dynamodb.GetItemOutput{Item: av}, nil)

	ok, err := s.repo.Exists(s.ctx, "+11234567890")
	s.Require().NoError(err)
	s.True(ok)
}

func (s *MemberRepoSuite) TestExists_NotFound() {
	s.client.EXPECT().GetItem(mock.Anything, mock.Anything).
		Return(&dynamodb.GetItemOutput{}, nil)

	ok, err := s.repo.Exists(s.ctx, "+10000000000")
	s.Require().NoError(err)
	s.False(ok)
}

func (s *MemberRepoSuite) TestExists_PropagatesError() {
	s.client.EXPECT().GetItem(mock.Anything, mock.Anything).
		Return(nil, errors.New("boom"))

	_, err := s.repo.Exists(s.ctx, "+11234567890")
	s.Require().Error(err)
}

func (s *MemberRepoSuite) TestGet_NotFoundReturnsErrNotFound() {
	s.client.EXPECT().GetItem(mock.Anything, mock.Anything).
		Return(&dynamodb.GetItemOutput{}, nil)

	mem, err := s.repo.Get(s.ctx, "+10000000000")
	s.Require().ErrorIs(err, repository.ErrNotFound)
	s.Nil(mem)
}

func TestMemberRepoSuite(t *testing.T) {
	suite.Run(t, new(MemberRepoSuite))
}
