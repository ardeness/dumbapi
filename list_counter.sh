#!/bin/bash
for x in $(curl -s http://localhost/counter/); do
	curl http://localhost/counter/${x}/
done
