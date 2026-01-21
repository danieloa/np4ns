#!/bin/bash
# Verification script for np4ns operator
# This script helps verify that the operator is working correctly

set -e

NAMESPACE="${1:-test-app-1}"
NP_NAME="enforced-network-policy"

echo "======================================"
echo "np4ns Operator Verification Script"
echo "======================================"
echo ""

# Check if namespace exists
echo "[1/6] Checking if namespace '$NAMESPACE' exists..."
if kubectl get namespace "$NAMESPACE" &>/dev/null; then
    echo "✅ Namespace '$NAMESPACE' exists"
else
    echo "❌ Namespace '$NAMESPACE' does not exist"
    echo "Creating namespace..."
    kubectl create namespace "$NAMESPACE"
    echo "✅ Namespace '$NAMESPACE' created"
fi
echo ""

# Wait a few seconds for operator to reconcile
echo "[2/6] Waiting 5 seconds for operator to reconcile..."
sleep 5
echo ""

# Check if network policy was created
echo "[3/6] Checking if enforced network policy exists..."
if kubectl get networkpolicy "$NP_NAME" -n "$NAMESPACE" &>/dev/null; then
    echo "✅ NetworkPolicy '$NP_NAME' exists in namespace '$NAMESPACE'"
else
    echo "❌ NetworkPolicy '$NP_NAME' does not exist in namespace '$NAMESPACE'"
    echo ""
    echo "Possible issues:"
    echo "  - Operator is not running"
    echo "  - Namespace is in the exception list"
    echo "  - Namespace is not in the target list (if NS_TARGET_FOR_NP is set)"
    exit 1
fi
echo ""

# Show the network policy spec
echo "[4/6] NetworkPolicy details:"
kubectl get networkpolicy "$NP_NAME" -n "$NAMESPACE" -o yaml | grep -A 30 "spec:"
echo ""

# Check namespace annotation
echo "[5/6] Checking namespace annotation..."
ANNOTATION=$(kubectl get namespace "$NAMESPACE" -o jsonpath='{.metadata.annotations.network-policy/enforced}')
if [ -n "$ANNOTATION" ]; then
    echo "✅ Namespace annotation 'network-policy/enforced': $ANNOTATION"
else
    echo "⚠️  Namespace annotation not found (might be added on next reconciliation)"
fi
echo ""

# Test deletion and recreation
echo "[6/6] Testing deletion and recreation..."
echo "Deleting network policy..."
kubectl delete networkpolicy "$NP_NAME" -n "$NAMESPACE"
echo "Waiting 10 seconds for operator to recreate..."
sleep 10

if kubectl get networkpolicy "$NP_NAME" -n "$NAMESPACE" &>/dev/null; then
    echo "✅ NetworkPolicy was successfully recreated by operator"
else
    echo "❌ NetworkPolicy was NOT recreated (operator might not be watching deletions)"
    exit 1
fi
echo ""

echo "======================================"
echo "✅ All verification checks passed!"
echo "======================================"
echo ""
echo "The np4ns operator is working correctly in namespace '$NAMESPACE'"
