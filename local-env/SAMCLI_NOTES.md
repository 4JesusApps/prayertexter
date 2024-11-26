Prereq:
1. docker
2. make
3. aws-cli
4. sam-cli

Compile:
1. make build
2. sudo sam local start-api --docker-network sam-backend
3. curl -d '{"body": "PLEASE PRAY FOR ME!", "phone-number":"777-777-7777"}' -H 'Content-Type: application/json' http://127.0.0.1:3000/

Start ddb and create tables:
1. ./local-env/dynamodb-setup.sh 

Good dynamodb commands:
1. aws dynamodb list-tables --endpoint-url http://localhost:8000
2. aws dynamodb execute-statement --statement "select * from Users" --endpoint-url http://localhost:8000