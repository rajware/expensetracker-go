#!/bin/sh
set -e

VARIANT=$1
VERSION=$2
OUTPUT=$3

if [ -z "$VARIANT" ] || [ -z "$VERSION" ] || [ -z "$OUTPUT" ]; then
    echo  "Usage: $0 VARIANT VERSION OUTPUTFILE"
    exit 1
fi

cat deploy/compose/${VARIANT}.yaml \
 | sed "s|image: quay.io/rajware/expensetracker-go:latest|image: quay.io/rajware/expensetracker-go:${VERSION}|g" \
 > ${OUTPUT}
