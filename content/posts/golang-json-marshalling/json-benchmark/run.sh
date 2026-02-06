#!/bin/bash

go test -bench=. -benchmem -benchtime=2s > bench.txt 2>&1

[ $? -ne 0 ] && cat bench.txt && exit 1

cat bench.txt | grep "Benchmark" | grep -v "Parse_SimdJson" | column -t

echo
echo "=== MARSHAL ==="
cat bench.txt | grep "Marshal" | awk '{print $1, $3, $5, $7}' | sed 's/Benchmark//g' | sed 's/-8//g' | column -t

echo
echo "=== UNMARSHAL ==="
cat bench.txt | grep "Unmarshal" | awk '{print $1, $3, $5, $7}' | sed 's/Benchmark//g' | sed 's/-8//g' | column -t
