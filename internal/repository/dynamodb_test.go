package repository_test

import (
	"context"
	"testing"

	"github.com/4JesusApps/prayertexter/internal/domain"
	"github.com/4JesusApps/prayertexter/internal/repository"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "John Doe", mem.Name)
	assert.Equal(s.T(), "+11234567890", mem.Phone)
}

func (s *DynamoDBRepoSuite) TestGet_NotFound() {
	s.client.EXPECT().
		GetItem(mock.Anything, mock.Anything).
		Return(&dynamodb.GetItemOutput{}, nil)

	mem, err := s.repo.Get(s.ctx, "+10000000000")
	require.NoError(s.T(), err)
	assert.Empty(s.T(), mem.Phone)
}

func (s *DynamoDBRepoSuite) TestSave_Success() {
	s.client.EXPECT().
		PutItem(mock.Anything, mock.Anything).
		Return(&dynamodb.PutItemOutput{}, nil)

	mem := &domain.Member{Phone: "+11234567890", Name: "Jane"}
	err := s.repo.Save(s.ctx, mem)
	require.NoError(s.T(), err)
}

func (s *DynamoDBRepoSuite) TestDelete_Success() {
	s.client.EXPECT().
		DeleteItem(mock.Anything, mock.Anything).
		Return(&dynamodb.DeleteItemOutput{}, nil)

	err := s.repo.Delete(s.ctx, "+11234567890")
	require.NoError(s.T(), err)
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
	require.NoError(s.T(), err)
	assert.Len(s.T(), members, 2)
	assert.Equal(s.T(), "A", members[0].Name)
	assert.Equal(s.T(), "B", members[1].Name)
}

func TestDynamoDBRepoSuite(t *testing.T) {
	suite.Run(t, new(DynamoDBRepoSuite))
}
