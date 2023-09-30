#!/bin/bash
set -e

cwd=$(dirname "$0")
project_root=$(dirname "$cwd")
envrc_file=$project_root/.envrc

echo "Check if necessary tools are installed"
command -v docker >/dev/null 2>&1 || { echo >&2 "docker is not installed. Aborting."; exit 1; }
command -v docker-compose >/dev/null 2>&1 || { echo >&2 "docker-compose is not installed. Aborting."; exit 1; }
command -v awslocal >/dev/null 2>&1 || { echo >&2 "awslocal is not installed. Aborting. Please run
  pip install awscli-local  "; exit 1; }

echo "Creating volume directory"
mkdir -p $cwd/_volume

echo "Starting localstack"
docker-compose -f $cwd/compose.yml up -d

echo "Waiting for localstack to be ready"
until awslocal sqs list-queues; do
  sleep 1
done

echo "Creating SQS queue"
queue_url=$(awslocal sqs create-queue --queue-name localstack-queue | jq -r '.QueueUrl')


echo "Setting environment variables in .env file"
echo "export AWS_SECRET_ACCESS_KEY=123" > $envrc_file
echo "export AWS_ACCESS_KEY_ID=123" >> $envrc_file
echo "export AWS_DEFAULT_REGION=us-east-1" >> $envrc_file
echo "export AWS_ENDPOINT_URL=http://localhost:4566" >> $envrc_file
echo "export OPENSEARCH_ADDRESS=http://localhost:9200" >> $envrc_file
echo "export OPENSEARCH_USERNAME=admin" >> $envrc_file
echo "export AWS_SQS_QUEUE_URL=$queue_url" >> $envrc_file

echo "Localstack is UI is at port"
echo "http://localhost:8080"

echo "Streaming logs"
docker-compose -f $cwd/compose.yml logs -f
