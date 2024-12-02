Prerequisites:
1. docker
2. make
3. aws-cli
4. sam-cli

Compile:
1. make build

Start local services and create tables:
1. sudo sam local start-api --docker-network sam-backend
2. ./local-env/dynamodb-setup.sh 

Test: 
1. curl http://127.0.0.1:3000/ -H 'Content-Type: application/json' -d '{"phone-number":"777-777-7777", "body": "PLEASE PRAY FOR ME!"}'

Good dynamodb commands:
1. aws dynamodb list-tables --endpoint-url http://localhost:8000
2. for table in ActivePrayers General Intercessors Members; do echo $table; aws dynamodb execute-statement --statement "select * from $table" --endpoint-url http://localhost:8000; echo; done