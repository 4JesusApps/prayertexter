package repository_test

import (
	"context"
	"testing"

	"github.com/4JesusApps/prayertexter/internal/domain"
	"github.com/4JesusApps/prayertexter/internal/repository"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	repomocks "github.com/4JesusApps/prayertexter/internal/mocks/repository"
)

type PrayerRepoSuite struct {
	suite.Suite
	client *repomocks.MockDDBClient
	repo   repository.PrayerRepository
	ctx    context.Context
}

func (s *PrayerRepoSuite) SetupTest() {
	s.client = repomocks.NewMockDDBClient(s.T())
	s.repo = repository.NewPrayerRepository(s.client, "ActivePrayer", "QueuedPrayer", 60)
	s.ctx = context.Background()
}

func tableNameMatcher(want string) any {
	return mock.MatchedBy(func(input any) bool {
		switch v := input.(type) {
		case *dynamodb.GetItemInput:
			return v.TableName != nil && *v.TableName == want
		case *dynamodb.PutItemInput:
			return v.TableName != nil && *v.TableName == want
		case *dynamodb.DeleteItemInput:
			return v.TableName != nil && *v.TableName == want
		case *dynamodb.ScanInput:
			return v.TableName != nil && *v.TableName == want
		default:
			return false
		}
	})
}

func (s *PrayerRepoSuite) TestGet_ActiveTable() {
	av, _ := attributevalue.MarshalMap(&domain.Prayer{IntercessorPhone: "+11234567890"})
	s.client.EXPECT().GetItem(mock.Anything, tableNameMatcher("ActivePrayer")).
		Return(&dynamodb.GetItemOutput{Item: av}, nil)

	pryr, err := s.repo.Get(s.ctx, "+11234567890", false)
	s.Require().NoError(err)
	s.Equal("+11234567890", pryr.IntercessorPhone)
}

func (s *PrayerRepoSuite) TestGet_QueuedTable() {
	av, _ := attributevalue.MarshalMap(&domain.Prayer{IntercessorPhone: "queue-id-1"})
	s.client.EXPECT().GetItem(mock.Anything, tableNameMatcher("QueuedPrayer")).
		Return(&dynamodb.GetItemOutput{Item: av}, nil)

	pryr, err := s.repo.Get(s.ctx, "queue-id-1", true)
	s.Require().NoError(err)
	s.Equal("queue-id-1", pryr.IntercessorPhone)
}

func (s *PrayerRepoSuite) TestSave_RoutesByQueuedFlag() {
	s.client.EXPECT().PutItem(mock.Anything, tableNameMatcher("QueuedPrayer")).
		Return(&dynamodb.PutItemOutput{}, nil)

	err := s.repo.Save(s.ctx, &domain.Prayer{IntercessorPhone: "queue-id-1"}, true)
	s.Require().NoError(err)
}

func (s *PrayerRepoSuite) TestDelete_RoutesByQueuedFlag() {
	s.client.EXPECT().DeleteItem(mock.Anything, tableNameMatcher("ActivePrayer")).
		Return(&dynamodb.DeleteItemOutput{}, nil)

	err := s.repo.Delete(s.ctx, "+11234567890", false)
	s.Require().NoError(err)
}

func (s *PrayerRepoSuite) TestGetAll_RoutesByQueuedFlag() {
	s.client.EXPECT().Scan(mock.Anything, tableNameMatcher("QueuedPrayer")).
		Return(&dynamodb.ScanOutput{Items: []map[string]types.AttributeValue{}}, nil)

	prayers, err := s.repo.GetAll(s.ctx, true)
	s.Require().NoError(err)
	s.Empty(prayers)
}

func (s *PrayerRepoSuite) TestExists_QueriesActiveTableOnly() {
	av, _ := attributevalue.MarshalMap(&domain.Prayer{IntercessorPhone: "+11234567890"})
	s.client.EXPECT().GetItem(mock.Anything, tableNameMatcher("ActivePrayer")).
		Return(&dynamodb.GetItemOutput{Item: av}, nil)

	ok, err := s.repo.Exists(s.ctx, "+11234567890")
	s.Require().NoError(err)
	s.True(ok)
}

func (s *PrayerRepoSuite) TestExists_NotFound() {
	s.client.EXPECT().GetItem(mock.Anything, tableNameMatcher("ActivePrayer")).
		Return(&dynamodb.GetItemOutput{}, nil)

	ok, err := s.repo.Exists(s.ctx, "+10000000000")
	s.Require().NoError(err)
	s.False(ok)
}

func TestPrayerRepoSuite(t *testing.T) {
	suite.Run(t, new(PrayerRepoSuite))
}
