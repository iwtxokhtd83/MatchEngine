#!/bin/bash
# Create GitHub Issues for MatchEngine
# Usage: GITHUB_TOKEN=ghp_xxxx bash scripts/create_issues.sh

REPO="iwtxokhtd83/MatchEngine"
API="https://api.github.com/repos/$REPO/issues"
AUTH="Authorization: token $GITHUB_TOKEN"
ACCEPT="Accept: application/vnd.github.v3+json"

if [ -z "$GITHUB_TOKEN" ]; then
  echo "Error: Set GITHUB_TOKEN first"
  echo "  export GITHUB_TOKEN=ghp_xxxx"
  echo "  bash scripts/create_issues.sh"
  exit 1
fi

echo "=== Creating issues for $REPO ==="

# Issue 1
echo "[1/10] float64 precision..."
