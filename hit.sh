#!/bin/bash -e

for i in $(seq 1 100);
do
  curl --silent --show-error "$1" > /dev/null & 
done

wait
