#!/bin/sh
set -e

RED="\033[31m"
GREEN="\033[32m"
YELLOW="\033[33m"
NORMAL="\033[39m"

LINTER_VERSION=1.48.0

LINTER_BINDIR=$(go env GOPATH)/bin
LINTER_NAME=golangci-lint
LINTER_EXEC=$LINTER_BINDIR/$LINTER_NAME-${LINTER_VERSION}

# Check if linter is installed and up to date up to date
if [[ ! -f $LINTER_EXEC ]]; then

    printf "${YELLOW}⣿ Downloading ${NORMAL}${LINTER_NAME}...\n"
    TMPDIR=`mktemp -d`
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $TMPDIR v$LINTER_VERSION

    mv $TMPDIR/${LINTER_NAME} $LINTER_EXEC
    printf "${YELLOW}⣿ Installed ${NORMAL}${LINTER_NAME} as \"${LINTER_EXEC}\"\n"
fi

if [[ "$CI" == "true" ]]; then
    $LINTER_EXEC run --out-format checkstyle ./... > build/report.xml
else
    $LINTER_EXEC run ./...
fi
