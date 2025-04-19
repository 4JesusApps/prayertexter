# local development

# unit tests

To run tests:
1. go test ./...

To run linting:
1. sudo docker run --rm -v $(pwd):/app -v ~/.cache/golangci-lint/:/root/.cache -w /app golangci/golangci-lint:latest golangci-lint run

# sam local testing

SAM local testing is done by creating local resources (dynamodb, api gateway, lambda). Dynamodb is set up with docker 
and a local dynamodb image. Tables need to get created every time, which is automated with a bash script. Sam-cli is 
used to simulate api gateway and lambda.

One thing worth mentioning is that sam local testing is using a different architecture than in AWS production. SAM local 
is using API gateway to trigger the lambda functions, while in production SNS topics trigger the main lambda function. 
The reason why we are using API gateway for sam local testing is because it is much easier to work with. It allows for a 
quick and simple way to test locally before moving to AWS production, where testing is more complex and slower.

Prerequisite software installs:

1. docker
2. make
3. aws-cli
4. sam-cli

Log into aws-cli with dummy access key and ID (this is required for creating local dynamodb tables)

1. aws configure
AWS Access Key ID [None]: 1
AWS Secret Access Key [None]: 1
Default region name [us-west-1]: 
Default output format [json]: 

Mandatory current working directory:

Everything on this page depends on you being in the localdev folder. This folder is where testing binaries have to get 
compiled from and it also has the development version cloudformation template with api gateway (to enable easier 
testing).

1. cd localdev

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

Compiled binary gets created in /bin/subfolder and then copied to the main directory. A binary needs to be compiled and 
in the main directory for sam local to work. The binary will be named bootstrap.

Create ddb tables and start local services:

1. ./dynamodb/setup.sh 
2. sudo sam local start-api --docker-network sam-backend

Test:

1. curl http://127.0.0.1:3000/ -H 'Content-Type: application/json' -d '{"originationNumber":"+17777777777", "messageBody": "pray"}'
2. monitor sam local api logs to view text message response

Good dynamodb commands:

1. aws dynamodb list-tables --endpoint-url http://localhost:8000
2. for table in ActivePrayer General Member QueuedPrayer; do echo $table; aws dynamodb execute-statement --statement "select * from $table" --endpoint-url http://localhost:8000; echo; done