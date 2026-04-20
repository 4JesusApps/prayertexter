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

type DynamoDBRepoSuite struct {
	suite.Suite
	client *repomocks.MockDDBClient
	repo   *repository.DynamoDBRepository[domain.Member]
	ctx    context.Context
}

func (s *DynamoDBRepoSuite) SetupTest() {
	s.client = repomocks.NewMockDDBClient(s.T())
	s.repo = repository.NewDynamoDBRepository[domain.Member](s.client, "Member", "Phone", 60)
	s.ctx = context.Background()
}

func (s *DynamoDBRepoSuite) TestGet_Success() {
	av, _ := attributevalue.MarshalMap(&domain.Member{
		Phone: "+11234567890",
		Name:  "John Doe",
	})

	s.client.EXPECT().
		GetItem(mock.Anything, mock.Anything).
		Return(&dynamodb.GetItemOutput{Item: av}, nil)

	mem, err := s.repo.Get(s.ctx, "+11234567890")
	s.Require().NoError(err)
	s.Equal("John Doe", mem.Name)
	s.Equal("+11234567890", mem.Phone)
}

func (s *DynamoDBRepoSuite) TestGet_NotFound() {
	s.client.EXPECT().
		GetItem(mock.Anything, mock.Anything).
		Return(&dynamodb.GetItemOutput{}, nil)

	mem, err := s.repo.Get(s.ctx, "+10000000000")
	s.Require().NoError(err)
	s.Empty(mem.Phone)
}

func (s *DynamoDBRepoSuite) TestSave_Success() {
	s.client.EXPECT().
		PutItem(mock.Anything, mock.Anything).
		Return(&dynamodb.PutItemOutput{}, nil)

	mem := &domain.Member{Phone: "+11234567890", Name: "Jane"}
	err := s.repo.Save(s.ctx, mem)
	s.Require().NoError(err)
}

func (s *DynamoDBRepoSuite) TestDelete_Success() {
	s.client.EXPECT().
		DeleteItem(mock.Anything, mock.Anything).
		Return(&dynamodb.DeleteItemOutput{}, nil)

	err := s.repo.Delete(s.ctx, "+11234567890")
	s.Require().NoError(err)
}

func (s *DynamoDBRepoSuite) TestGetAll_Success() {
	mem1, _ := attributevalue.MarshalMap(&domain.Member{Phone: "+11111111111", Name: "A"})
	mem2, _ := attributevalue.MarshalMap(&domain.Member{Phone: "+12222222222", Name: "B"})

	s.client.EXPECT().
		Scan(mock.Anything, mock.Anything).
		Return(&dynamodb.ScanOutput{
			Items: []map[string]types.AttributeValue{mem1, mem2},
		}, nil)

	members, err := s.repo.GetAll(s.ctx)
	s.Require().NoError(err)
	s.Len(members, 2)
	s.Equal("A", members[0].Name)
	s.Equal("B", members[1].Name)
}

func TestDynamoDBRepoSuite(t *testing.T) {
	suite.Run(t, new(DynamoDBRepoSuite))
}
