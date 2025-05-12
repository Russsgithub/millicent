#!/bin/bash

for file in ./uploads/*; do
	results=$(python analyze.py "$file")
	curl -X POST -H "Content-Type: application/json" -d "$results" http://localhost:8080/content
	mv "$file" ./processed/
done