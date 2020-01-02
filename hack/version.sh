#!/bin/sh
# Simple shell script that parses git revisions and returns best matching semver tag for it
# Following rules are applied:
#  - If current commit is same as a latest tag - the tag name is returned. E.g "v1.0.2"
#  - If current commit does not match latest tag - a tag name with appended commit is returned, e.g "v1.0.2+a3dc218"
#  - If no tags have been created - current commit is appended to "v0.0.0", e.g "v0.0.0+a3dc218"
#

LATEST_TAG_REV=$(git rev-list --tags --max-count=1)
LATEST_COMMIT_REV=$(git rev-list HEAD --max-count=1)

if [ -n "$LATEST_TAG_REV" ]; then
    LATEST_TAG=$(git describe --tags "$(git rev-list --tags --max-count=1)")
else
    LATEST_TAG="v0.0.0"
fi

if [ "$LATEST_TAG_REV" != "$LATEST_COMMIT_REV" ]; then
    echo "$LATEST_TAG+$(git rev-list HEAD --max-count=1 --abbrev-commit)"
else
    echo "$LATEST_TAG"
fi
