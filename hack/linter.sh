#!/bin/sh
set -e

RED="\033[31m"
GREEN="\033[32m"
YELLOW="\033[33m"
NORMAL="\033[39m"

LINTER_VERSION=2.13.2

LINTER_BINDIR=$(go env GOPATH)/bin
LINTER_NAME=golangci-lint
LINTER_EXEC=$LINTER_BINDIR/$LINTER_NAME-${LINTER_VERSION}
# Official installer URL - the previously used master branch is no longer maintained
# and its copy of the script fails checksum verification (golangci-lint#6539)
LINTER_INSTALLER=https://golangci-lint.run/install.sh

# Check if linter is installed and up to date
if [[ ! -f $LINTER_EXEC ]]; then

    printf "${YELLOW}⣿ Downloading ${NORMAL}${LINTER_NAME}...\n"
    TMPDIR=$(mktemp -d)
    # The script is piped to a shell, so a redirect to plain HTTP would hand an attacker
    # arbitrary code execution - restrict both the request and any redirect to HTTPS
    curl --proto '=https' --proto-redir '=https' --tlsv1.2 -sSfL "$LINTER_INSTALLER" \
        | sh -s -- -b "$TMPDIR" "v$LINTER_VERSION"

    mv $TMPDIR/${LINTER_NAME} $LINTER_EXEC
    printf "${YELLOW}⣿ Installed ${NORMAL}${LINTER_NAME} as \"${LINTER_EXEC}\"\n"
fi

if [[ "$CI" == "true" ]]; then
    $LINTER_EXEC run --output.checkstyle.path build/report.xml ./...
else
    $LINTER_EXEC run ./...
fi
