#!/bin/bash

# Fix handler.go files - change to int64
sed -i '' 's/Total:  len(\([^)]*\))/Total:  int64(len(\1))/g' internal/payments/handler.go internal/notifications/handler.go

# Fix handler.go - use SuccessResponseWithMetaAndStatus
sed -i '' 's/common\.SuccessResponseWithMeta(c, http\.StatusOK, \([^,]*\), \&common\.Meta/common.SuccessResponseWithMeta(c, \1, \&common.Meta/g' internal/payments/handler.go internal/notifications/handler.go

# Remove the extra arguments from SuccessResponseWithMeta calls
sed -i '' 's/, ".*retrieved successfully")/, nil)/g' internal/payments/handler.go internal/notifications/handler.go

echo "Fixed basic issues. Now fixing logger calls..."

# This is complex, we'll do it manually for critical files
