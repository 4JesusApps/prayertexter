# prayertexter

This application is a work in progress!

prayertexter allows members to send in prayer requests to a specific phone number. Once a prayer request is received, it will get sent to multiple other members (Intercessors) who have signed up to pray for others. Once someone has prayed for a prayer request that they have received, they text back "prayed". This will alert the member who sent in the prayer request that their request has been prayed for.

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
1. make build-prayertexter

Create ddb tables and start local services:
1. ./localdev/dynamodbsetup.sh 
2. sudo sam local start-api --docker-network sam-backend

Test: 
1. curl http://127.0.0.1:3000/ -H 'Content-Type: application/json' -d '{"phone-number":"+17777777777", "body": "pray"}'
2. monitor sam local api logs to view text message response

Good dynamodb commands:
1. aws dynamodb list-tables --endpoint-url http://localhost:8000
2. for table in ActivePrayers General Members QueuedPrayers; do echo $table; aws dynamodb execute-statement --statement "select * from $table" --endpoint-url http://localhost:8000; echo; done

# TODO

- create reconciler that runs on interval periods which will check and fix inconsistencies
    - check prayer queue table and assign prayers if possible
    - some level of continue off of previous failures
    - check that all phones on intercessor phones list are for active members (not sure if needed, low priority, potential high ddb cost to run get on all intercessors)
    - check all active prayers have active intercessors (this would only be needed to recover from inconsistent states; possible low priority)
    - send out prayer reminder texts to intercessors after x number of hours with unprayed prayer requests
        - also consider copying prayer to another intercessor if it has not been prayed for in x number of hours
- long tests utilizing real ddb, lambda, sns, and sim phone numbers
    - implement simulator numbers with sns topics
    - implement secure way to save authentication
- rename state tracker to fault tracker - is tracking error states the only thing necessary? is there any benefit to track completed requests?
- unit test state tracker in real flow to verify errors are saved
- move 10-DLC number from sandbox to prod
- dynamodb pagination for IntercessorPhones and StateTracker due to possible long length - is it needed?
- validate phone number format - is there any benefit or is this overkill?
- implement dynamodb conditional updates for race conditions/concurrency safety (FindIntercessors, possibly others)
    - this may help with allowing for concurrent lambda functions to run
- decide if/where to implement dynamodb strongly consistent writes (as opposed to default eventual consistency)
    - this may help with allowing for concurrent lambda functions to run
- implement text/template for message content
- add tests for aws retry mechanism - is it worth it?
- add logging for aws retry operations - is it worth it?
- pass down lambda handler context to lower aws service functions so that context can be re-used
    - this will follow go best practices and allow for better aws debugging (xray tracing)
- save all initial sign up text messages for 10-DLC number requirements
- unit tests for config
- web frontend for sign up process
    - possibly could add other features eventually
    - minimum required feature is to be able to complete entire sign up flow in a web form and submit to be added to prayer texter app