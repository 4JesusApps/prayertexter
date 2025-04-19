#!/usr/bin/env bash

sudo docker compose -f dynamodb/compose.yaml down
sudo docker compose -f dynamodb/compose.yaml up -d
sleep 5
aws dynamodb create-table --cli-input-json file://dynamodb/activeprayer-table.json --endpoint-url http://localhost:8000
aws dynamodb create-table --cli-input-json file://dynamodb/general-table.json --endpoint-url http://localhost:8000
aws dynamodb create-table --cli-input-json file://dynamodb/member-table.json --endpoint-url http://localhost:8000
aws dynamodb create-table --cli-input-json file://dynamodb/queuedprayer-table.json --endpoint-url http://localhost:8000