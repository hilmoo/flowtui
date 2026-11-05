#!/bin/sh
# Appends three sample requests to the history file, then succeeds.
mkdir -p out
now=$(date -u +%Y-%m-%dT%H:%M:%SZ)
echo "hello flow starting"
cat >> out/history.jsonl <<JSON
{"time":"$now","method":"POST","url":"http://localhost:8080/api/ProjectCreate","request":{"name":"demo","tags":["a","b"]},"status":201,"durationMs":42,"response":{"id":7,"name":"demo","createdAt":"$now"}}
{"time":"$now","method":"GET","url":"http://localhost:8080/api/ProjectList?page=1","status":200,"durationMs":12,"response":{"items":[{"id":7,"name":"demo"}],"total":1,"screenshot":"$(printf 'A%.0s' $(seq 1 400))"}}
{"time":"$now","method":"GET","url":"http://localhost:8080/api/Missing","status":404,"durationMs":5,"response":"Not Found"}
JSON
echo "hello flow done"
