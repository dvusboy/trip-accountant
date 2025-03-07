#!/bin/sh

# This script starts a container of the trip-accountant image,
# and uses "curl" to make some REST API calls against it.
# It only tests happy-paths and doesn't validate the returned
# JSON messages.
# At the end, the container is stopped and cleaned up.

set -e

image_tag=$1

# Start the container
docker run -d --name=trip-accountant -p 127.0.0.1:8081:8081 $image_tag
sleep 1
# Create trip
curl -v http://127.0.0.1:8081/trips -H 'context-type: application/json' -d '{"owner":"alice@test.com", "name":"Test trip", "start_date":"2025-01-01", "description":"Testing a fun trip", "participants":["bob@test.com", "charlie@test.com"]}'
echo
# Load trips
curl -v http://127.0.0.1:8081/alice@test.com/trips
echo
# Add Expense
curl -v http://127.0.0.1:8081/trips/1/expenses -H "content-type: application/json" -d '{"date":"2025-01-02", "description":"tickets", "participants":{"alice@test.com":6000, "bob@test.com":0, "charlie@test.com":0}}'
echo
# Settle
curl -v http://127.0.0.1:8081/trips/1/settlement
echo
# Shutdown container
docker stop trip-accountant && docker rm -v trip-accountant
