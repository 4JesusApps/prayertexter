Prerequisites:
1. docker
2. make
3. aws-cli
4. sam-cli

Compile:
1. make build

Create ddb tables and start local services:
1. ./local-env/dynamodb-setup.sh 
2. sudo sam local start-api --docker-network sam-backend

Test: 
1. curl http://127.0.0.1:3000/ -H 'Content-Type: application/json' -d '{"phone-number":"777-777-7777", "body": "PLEASE PRAY FOR ME!"}'

Good dynamodb commands:
1. aws dynamodb list-tables --endpoint-url http://localhost:8000
2. for table in ActivePrayers General Intercessors Members; do echo $table; aws dynamodb execute-statement --statement "select * from $table" --endpoint-url http://localhost:8000; echo; done

TODO:
- prevent situation where prayer has 0 available intercessors (most likely because they have active prayers)
- prevent infinite look when looking for intercessors (if none)
- prevent lambda from running multiple times due to failure (return from main func???, pass errors upwards???)
- idempotency with potential for multiple calls with same data
- add check into find intercessors that checks their number in active prayers
- separate marshal and unmarshal map out into separate functions (for member and prayer) so they can be used in unit testing and not duplicated