#!/bin/bash

counterids=$(curl http://localhost/counter)

for id in $(curl -s http://localhost/counter/);
do
	curl -X POST "http://localhost/counter/${id}/stop/"
done
