#!/bin/bash

if [ -n "$1" ]; then
  TAG_NAME=$1
else
  echo "Error: No Git tags found and no TAG_NAME provided. Exiting."
  exit 1
fi

export TAG_NAME=$TAG_NAME
docker compose down
docker compose up -d
