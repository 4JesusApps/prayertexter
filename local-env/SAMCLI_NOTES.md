1. cd OneDrive/Documents/Personal/4JesusApps/CODE/
2. Modified template - see Medium + aws link
3. GOARCH=amd64 GOOS=linux go build -o bootstrap main.go
4. sudo sam local start-api --docker-network sam-backend
5. curl -d '{"body": "PLEASE PRAY FOR ME!", "phone-number":"777-777-7777"}' -H 'Content-Type: application/json' http://127.0.0.1:3000/

1. Start ddb and create tables:
./local-env/dynamodb-setup.sh 

1. Good dynamodb commands:
aws dynamodb list-tables --endpoint-url http://localhost:8000
aws dynamodb execute-statement --statement "select * from Users" --endpoint-url http://localhost:8000