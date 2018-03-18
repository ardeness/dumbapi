#!/bin/bash

numofcounter=$(curl -s localhost/counter/ | wc -l)

until [[ ! $numofcounter -gt 0 ]]
do
	sleep 1
	numofcounter=$(curl -s localhost/counter/ | wc -l)
done
