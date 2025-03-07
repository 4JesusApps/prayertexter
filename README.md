# prayer-texter

This application is a work in progress!

Prayer Texter allows members to send in prayer requests to a specific phone number. Once a prayer request is received, it will get sent to multiple other members (Intercessors) who have signed up to pray for others. Once someone has prayed for a prayer request that they have received, they text back "prayed". This will alert the member who sent in the prayer request that their request has been prayed for.

# unit tests

You can add the following environmental variable to your linux session when running unit tests and it will log every text message response. This can be helpful when running unit tests to see all text messages sent out prior to some
unexpected error, as it will generally give you a hint on what is going wrong right before the failure.
1. export AWS_SAM_LOCAL=true

# sam local testing

SAM local testing is done by creating local resources (dynamodb, api gateway, lambda). Dynamodb is set up with docker and a local dynamodb image.
Tables need to get created every time, which is automated with a bash script. Sam-cli is used to simulate api gateway and lambda.

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
1. curl http://127.0.0.1:3000/ -H 'Content-Type: application/json' -d '{"phone-number":"+17777777777", "body": "PLEASE PRAY FOR ME!"}'
2. monitor sam local api logs to view text message response

Good dynamodb commands:
1. aws dynamodb list-tables --endpoint-url http://localhost:8000
2. for table in ActivePrayers General Members PrayersQueue; do echo $table; aws dynamodb execute-statement --statement "select * from $table" --endpoint-url http://localhost:8000; echo; done

# TODO

- create reconciler that runs on interval periods which will check and fix inconsistencies
    - check prayer queue table and assign prayers if possible
    - some level of continue off of previous failures
    - check that all phones on intercessor phones list are for active members (maybe, low priority, potential high ddb cost to run get on all intercessors)
    - check all active prayers have active intercessors (this would only be needed to recover from inconsistent states; possible low priority)
- long tests utilizing real ddb, lambda, sns, and sim phone numbers
    - implement simulator numbers with sns topics
    - implement secure way to save authentication
- rename state tracker to fault tracker???
- unit test state tracker in real flow to verify errors are saved
- improve error handling; try to log at lowest level to reduce log code
    - review all functions/methods properly return each error
    - find way to log error path so logging at top level is unnecessary
- move 10-DLC number from sandbox to prod