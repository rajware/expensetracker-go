#!/bin/sh
set -e

VARIANT=$1
VERSION=$2
OUTPUT=$3

if [ -z "$VARIANT" ] || [ -z "$VERSION" ] || [ -z "$OUTPUT" ]; then
    echo  "Usage: $0 VARIANT VERSION OUTPUTFILE"
    exit 1
fi

VARIANT_DIR="deploy/kubernetes/${VARIANT}"
VARIANT_DOCFILE="${VARIANT_DIR}/doc.txt"
VARIANT_FILES=
case $VARIANT in
  tracker-sqlite)
    VARIANT_FILES="pvc secret dep svc ingress"
    ;;
  tracker-postgres)
    VARIANT_FILES="pvc secret db-dep db-svc fe-dep fe-svc ingress"
    ;;
  *)
    echo "Invalid variant: ${VARIANT}"
    exit 1
esac

if [ -f "${VARIANT_DOCFILE}" ]; then
    cat "${VARIANT_DOCFILE}" > "$OUTPUT"
else
    # If doc.txt is missing, ensure the output file starts empty
    : > "$OUTPUT"
fi

for i in ${VARIANT_FILES}; do 
    printf -- "---\n"
    cat deploy/kubernetes/${VARIANT}/${VARIANT}-$i.yaml
done \
 | sed "1d; s|image: quay.io/rajware/expensetracker-go:latest|image: quay.io/rajware/expensetracker-go:${VERSION}|g" \
 >> ${OUTPUT}
