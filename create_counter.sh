#!/bin/bash
for x in `seq 1 100`; do
	curl -X POST "http://localhost/counter/?to=$(((RANDOM%1000)+1000))";
done
