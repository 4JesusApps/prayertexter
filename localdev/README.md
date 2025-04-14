# local development

# unit tests

To run tests:
1. go test ./...

To run linting:
1. sudo docker run --rm -v $(pwd):/app -v ~/.cache/golangci-lint/:/root/.cache -w /app golangci/golangci-lint:latest golangci-lint run

# sam local testing

SAM local testing is done by creating local resources (dynamodb, api gateway, lambda). Dynamodb is set up with docker and a local dynamodb image.
Tables need to get created every time, which is automated with a bash script. Sam-cli is used to simulate api gateway and lambda.

Prerequisites:

1. docker
2. make
3. aws-cli
4. sam-cli

Compile:

There are several different options when compiling. To run local tests, you have to compile and have a bootstrap binary
in the current working directory.

1. Compile prayertexter app with default x64 cpu architecture:
make build
2. Compile prayertexter app with arm64 cpu architecture:
make build ARCH=arm64
3. Compile other-than prayertexter binary on arm64 cpu architecture:
make build BUILD=announcer ARCH=arm64
4. Delete /bin and bootstrap binary in current dir:
make clean

Compiled binary gets created in /bin/subfolder and then copied to to the current directory.

Create ddb tables and start local services:

1. ./localdev/dynamodbsetup.sh 
2. sudo sam local start-api --docker-network sam-backend

Test:

1. curl http://127.0.0.1:3000/ -H 'Content-Type: application/json' -d '{"phone-number":"+17777777777", "body": "pray"}'
2. monitor sam local api logs to view text message response

Good dynamodb commands:

1. aws dynamodb list-tables --endpoint-url http://localhost:8000
2. for table in ActivePrayers General Members QueuedPrayers; do echo $table; aws dynamodb execute-statement --statement "select * from $table" --endpoint-url http://localhost:8000; echo; done