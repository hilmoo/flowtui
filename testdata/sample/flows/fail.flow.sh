#!/bin/sh
# Records one failing request, prints to stderr and exits non-zero.
mkdir -p out
now=$(date -u +%Y-%m-%dT%H:%M:%SZ)
echo "fail flow starting"
echo '{"time":"'"$now"'","method":"PUT","url":"http://localhost:8080/api/ProjectUpdate","request":{"id":7},"status":500,"durationMs":130,"response":{"error":"boom"}}' >> out/history.jsonl
echo "this is not json" >> out/history.jsonl
echo "something went wrong" >&2
exit 3
