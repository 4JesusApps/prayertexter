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

# TODO

- create reconciler that runs on interval periods which will check and fix inconsistencies
    - check prayer queue table and assign prayers if possible
        - send message to let requestor know that their prayer got sent out
    - some level of retry from previous failures
        - stages that need special retry logic
            - sign up stage - if error happens and user works past it by retrying themselves, we do not want to revert
              a members sign up status behind that which is currently is (ie changing stage from 3 to 2)
        - stages that can be retried without issues:
            - member delete
        - might be OK to retry even with side affects:
            - prayer request - more prayers being assigned to intercessors, or queued, possible intercessors getting +1
              prayer count without being assigned prayer
        - need to consider how to track sent text messages so we can avoid resending all previously sent messages
            - maybe statetracker can have a count of # of messages sent which would provide a decent indication whether
              or not messages need to get sent on retry
    - check that all phones on intercessor phones list are for active members (not sure if needed, low priority, potential high ddb cost to run get on all intercessors)
    - check all active prayers have active intercessors (this would only be needed to recover from inconsistent states; possible low priority)
    - send out prayer reminder texts to intercessors after x number of hours with unprayed prayer requests
        - also consider copying prayer to another intercessor if it has not been prayed for in x number of hours
- tests utilizing real ddb, lambda, sns, and sim phone numbers
    - implement simulator numbers with sns topics
    - implement secure way to save authentication
- rename state tracker to fault tracker - is tracking error states the only thing necessary? is there any benefit to track completed requests?
- unit test state tracker in real flow to verify errors are saved
- move 10-DLC number from sandbox to prod
- implement dynamodb conditional updates for race conditions/concurrency safety (FindIntercessors, possibly others)
    - this may help with allowing for concurrent lambda functions to run
- decide if/where to implement dynamodb strongly consistent writes (as opposed to default eventual consistency)
    - this may help with allowing for concurrent lambda functions to run
- decide if/where to implement dynamodb optimistic locking
    - this may help with allowing for concurrent lambda functions to run
- save all initial sign up text messages for 10-DLC number requirements
- web frontend for sign up process
    - possibly could add other features eventually
    - minimum required feature is to be able to complete entire sign up flow in a web form and submit to be added to prayer texter app
- dynamodb TransactWriteItems for atomic write operations (to help with error recovery)
    - this will complicate current implementation of dynamodb because new functions and mocks will be needed
    - this will allow for better recovery because TransactWriteItems allows for grouping of multiple put/delete items into a single
      transaction so that if one fails, they all fail
    - the goal is for atomic write operations to help with easier retry; retry would not have to consider half completed
      operations
    - implementation details:
        - stages that might need atomic write operations:
            - prayer request (if not ok with retry side effects, which aren't that bad)
        - stages that do not need:
            - sign up
            - member delete
            - complete prayer
- implement "sorry there was a problem, please try again later" text message if issues occur