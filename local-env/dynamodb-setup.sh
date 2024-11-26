#!/usr/bin/env bash

cd ~/OneDrive/Documents/Personal/4JesusApps/CODE/local-env
sudo docker ps | grep dynamodb && {
    sudo docker compose down
}
sudo docker compose up -d
sleep 5
aws dynamodb create-table --cli-input-json file://active-prayers-table.json --endpoint-url http://localhost:8000
aws dynamodb create-table --cli-input-json file://intercessors-table.json --endpoint-url http://localhost:8000
aws dynamodb create-table --cli-input-json file://members-table.json --endpoint-url http://localhost:8000