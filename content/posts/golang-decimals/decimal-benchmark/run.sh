#!/bin/bash

go test -bench=. -benchmem -benchtime=2s > bench.txt 2>&1

[ $? -ne 0 ] && cat bench.txt && exit 1

echo "=== ADDITION ==="
cat bench.txt | grep "Addition" | awk '{print $1, $3, $5, $7}' | sed 's/Benchmark//g' | sed 's/-8//g' | column -t

echo
echo "=== MULTIPLICATION ==="
cat bench.txt | grep "Multiply" | awk '{print $1, $3, $5, $7}' | sed 's/Benchmark//g' | sed 's/-8//g' | column -t

echo
echo "=== DIVISION ==="
cat bench.txt | grep "Divide" | awk '{print $1, $3, $5, $7}' | sed 's/Benchmark//g' | sed 's/-8//g' | column -t

echo
echo "=== PARSING ==="
cat bench.txt | grep "Parse" | awk '{print $1, $3, $5, $7}' | sed 's/Benchmark//g' | sed 's/-8//g' | column -t

echo
echo "=== COMPLEX CALCULATION ==="
cat bench.txt | grep "Complex" | awk '{print $1, $3, $5, $7}' | sed 's/Benchmark//g' | sed 's/-8//g' | column -t
