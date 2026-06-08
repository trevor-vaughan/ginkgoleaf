#!/usr/bin/env bash
# Verify that a consumer importing only `ginkgoleaf` (and not the opt-in
# `ginkgoleaf/gomega` sub-package) does NOT acquire gomega as a direct
# or indirect dependency.
#
# We probe via fixtures/no-gomega's go.mod. go.sum is NOT the right
# check: it records hashes for the entire transitive graph (including
# gomega via ginkgo's own go.mod) for cryptographic verification, which
# is independent of what the consumer actually depends on.
set -euo pipefail

GO_MOD="fixtures/no-gomega/go.mod"

if [[ ! -f "$GO_MOD" ]]; then
	echo "FAIL: $GO_MOD does not exist; run 'go mod tidy' in fixtures/no-gomega first" >&2
	exit 1
fi

if grep -qE 'onsi/gomega' "$GO_MOD"; then
	echo "FAIL: gomega leaked into $GO_MOD (a no-gomega consumer should not declare gomega):" >&2
	grep -E 'onsi/gomega' "$GO_MOD" >&2
	exit 1
fi

echo "OK: gomega not present in $GO_MOD"
