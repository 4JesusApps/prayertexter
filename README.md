# prayer-texter

This application is a work in progress!

Prayer Texter allows members to send in prayer requests to a specific phone number. Once a prayer request is received, it will get sent to multiple other members (Intercessors) who have signed up to pray for others. Once someone has prayed for a prayer request that they have received, they text back "prayed". This will alert the member who sent in the prayer request that their request has been prayed for.

# local testing

Local testing is done by creating local resources (dynamodb, api gateway, lambda). Dynamodb is set up with docker and a local dynamodb image.
Tables need to get created manually every time, which is automated with a bash script. Sam-cli is used to simulate api gateway and lambda.

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
2. for table in ActivePrayers General Members PrayersQueue; do echo $table; aws dynamodb execute-statement --statement "select * from $table" --endpoint-url http://localhost:8000; echo; done

# TODO

- create reconciler that runs on interval periods which will check and fix inconsistencies
    - check prayer queue table and assign prayers if possible
    - some level of continue off of previous failures
    - check that all phones on intercessor phones list are for active members (maybe, low priority, potential high ddb cost to run get on all intercessors)
    - clear out old states from state tracker
- add unit test for mem.checkIfActive(clnt)
- if user cancels and they are Intercessor with active prayer, move prayer to someone else
- implement 10DLC phone number
- implement unit tests for non-main files
- prevent intercessors from praying for their own request
- long tests utilizing local ddb and sam local apigateway/lambda