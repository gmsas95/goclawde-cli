#!/bin/bash
# Smoke Tests for Myrai
# Quick verification that the binary builds and basic features work

BINARY="./bin/myrai"
PASS=0
FAIL=0

echo "=========================================="
echo "  Myrai Smoke Tests"
echo "=========================================="
echo

# Build
echo "[1/8] Building binary..."
if go build -o "$BINARY" ./cmd/myrai 2>&1; then
    echo "  ✅ Build successful"
    ((PASS++))
else
    echo "  ❌ Build failed"
    ((FAIL++))
    exit 1
fi

# Test --help
echo "[2/8] Testing --help..."
OUTPUT=$($BINARY --help < /dev/null 2>&1)
if echo "$OUTPUT" | grep -q "Myrai - Your Personal AI Assistant"; then
    echo "  ✅ --help works"
    ((PASS++))
else
    echo "  ❌ --help failed"
    ((FAIL++))
fi

# Test version
echo "[3/8] Testing version..."
OUTPUT=$($BINARY version < /dev/null 2>&1)
if echo "$OUTPUT" | grep -q "Myrai version"; then
    echo "  ✅ version works"
    ((PASS++))
else
    echo "  ❌ version failed"
    ((FAIL++))
fi

# Test help subcommand
echo "[4/8] Testing help subcommand..."
OUTPUT=$($BINARY help < /dev/null 2>&1)
if echo "$OUTPUT" | grep -q "Usage:"; then
    echo "  ✅ help subcommand works"
    ((PASS++))
else
    echo "  ❌ help subcommand failed"
    ((FAIL++))
fi

# Test project help
echo "[5/8] Testing project help..."
OUTPUT=$($BINARY project < /dev/null 2>&1)
if echo "$OUTPUT" | grep -q "Project Management"; then
    echo "  ✅ project help works"
    ((PASS++))
else
    echo "  ❌ project help failed"
    ((FAIL++))
fi

# Test batch help
echo "[6/8] Testing batch help..."
OUTPUT=$($BINARY batch -h < /dev/null 2>&1)
if echo "$OUTPUT" | grep -q "Batch Processing"; then
    echo "  ✅ batch help works"
    ((PASS++))
else
    echo "  ❌ batch help failed"
    ((FAIL++))
fi

# Test config help
echo "[7/8] Testing config help..."
OUTPUT=$($BINARY config < /dev/null 2>&1)
if echo "$OUTPUT" | grep -q "Config Commands"; then
    echo "  ✅ config help works"
    ((PASS++))
else
    echo "  ❌ config help failed"
    ((FAIL++))
fi

# Test doctor (should work even without config)
echo "[8/8] Testing doctor..."
OUTPUT=$($BINARY doctor < /dev/null 2>&1)
if echo "$OUTPUT" | grep -q "Diagnostics"; then
    echo "  ✅ doctor works"
    ((PASS++))
else
    echo "  ❌ doctor failed"
    ((FAIL++))
fi

echo
echo "=========================================="
echo "  Results: $PASS passed, $FAIL failed"
echo "=========================================="

if [ $FAIL -gt 0 ]; then
    exit 1
fi
