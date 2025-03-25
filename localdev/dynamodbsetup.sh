#!/usr/bin/env bash

cd localdev || exit
sudo docker compose down
sudo docker compose up -d
sleep 5
aws dynamodb create-table --cli-input-json file://active-prayers-table.json --endpoint-url http://localhost:8000
aws dynamodb create-table --cli-input-json file://general-table.json --endpoint-url http://localhost:8000
aws dynamodb create-table --cli-input-json file://members-table.json --endpoint-url http://localhost:8000
aws dynamodb create-table --cli-input-json file://queued-prayers-table.json --endpoint-url http://localhost:8000