#!/usr/bin/env bash

sudo docker compose -f dev/dynamodb/compose.yaml down
sudo docker compose -f dev/dynamodb/compose.yaml up -d
sleep 5
aws dynamodb create-table --cli-input-json file://dev/dynamodb/activeprayer-table.json --endpoint-url http://localhost:8000
aws dynamodb create-table --cli-input-json file://dev/dynamodb/general-table.json --endpoint-url http://localhost:8000
aws dynamodb create-table --cli-input-json file://dev/dynamodb/member-table.json --endpoint-url http://localhost:8000
aws dynamodb create-table --cli-input-json file://dev/dynamodb/queuedprayer-table.json --endpoint-url http://localhost:8000