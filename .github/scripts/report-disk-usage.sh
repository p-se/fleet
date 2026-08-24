#!/bin/bash

# Reports runner disk usage at a labelled checkpoint, so CI failures caused by
# ephemeral-storage eviction can be traced back to what actually filled the
# disk. k3d nodes are containers sharing the runner's root filesystem, so
# kubelet's eviction signal reflects the whole runner, not any single cluster.
#
# Usage:
#   report-disk-usage.sh <label> [--deep]   record a checkpoint
#   report-disk-usage.sh --summary          print the trend table
#
# This script never fails the job.

set -u

label="${1:-unlabelled}"
deep="${2:-}"

report="${DISK_REPORT_FILE:-/tmp/disk-usage-report.txt}"
trend="${DISK_TREND_FILE:-/tmp/disk-usage-trend.csv}"

mkdir -p "$(dirname "$report")" "$(dirname "$trend")"

# Preinstalled tooling on GitHub-hosted runners. Everything here is a candidate
# for removal before provisioning clusters, none of it is used by Fleet's e2e.
PRUNE_CANDIDATES=(
  /usr/share/dotnet
  /usr/local/lib/android
  /opt/ghc
  /usr/local/.ghcup
  /usr/share/swift
  /usr/local/share/powershell
  /usr/local/share/boost
  /usr/share/miniconda
  /usr/lib/google-cloud-sdk
  /opt/az
  /opt/microsoft
  /usr/local/julia*
  /usr/local/lib/node_modules
  /opt/hostedtoolcache/CodeQL
  /opt/hostedtoolcache/PyPy
  /opt/hostedtoolcache/Ruby
)

mb() { # size of a path in MB, staying on one filesystem, 0 when missing
  local total=0 sz
  for p in $1; do
    [ -e "$p" ] || continue
    sz=$(sudo du -sxm "$p" 2>/dev/null | cut -f1)
    total=$((total + ${sz:-0}))
  done
  echo "$total"
}

emit() { echo "$*" | tee -a "$report"; }

if [ "$label" = "--summary" ]; then
  echo "::group::Disk usage trend"
  {
    printf '%-26s %10s %10s %8s\n' CHECKPOINT AVAIL_GB USED_GB USED_PCT
    while IFS=, read -r l avail used pct; do
      printf '%-26s %10s %10s %8s\n' "$l" "$avail" "$used" "$pct"
    done < "$trend"
  } | tee -a "${GITHUB_STEP_SUMMARY:-/dev/null}"
  echo "::endgroup::"
  exit 0
fi

avail_kb=$(df -Pk / | awk 'NR==2 {print $4}')
used_kb=$(df -Pk / | awk 'NR==2 {print $3}')
used_pct=$(df -Pk / | awk 'NR==2 {print $5}')
printf '%s,%.1f,%.1f,%s\n' "$label" \
  "$(echo "$avail_kb" | awk '{print $1/1048576}')" \
  "$(echo "$used_kb" | awk '{print $1/1048576}')" \
  "$used_pct" >> "$trend"

echo "::group::Disk usage: $label"
emit "===== checkpoint: $label ($(date -u +%FT%TZ)) ====="
emit ""
emit "--- filesystem ---"
df -h / | tee -a "$report"
df -i / | tee -a "$report"
emit ""
emit "k3s evicts at nodefs.available<5% of the root filesystem."
emit "eviction floor: $(df -Pk / | awk 'NR==2 {printf "%.2f GiB", ($2*0.05)/1048576}')"
emit "currently available: $(echo "$avail_kb" | awk '{printf "%.2f GiB", $1/1048576}')"
emit ""

emit "--- job-created (what our own workflow costs) ---"
for entry in \
  "workspace:${GITHUB_WORKSPACE:-$PWD}" \
  "docker:/var/lib/docker" \
  "go-build-cache:$(go env GOCACHE 2>/dev/null || echo /nonexistent)" \
  "go-mod-cache:$(go env GOMODCACHE 2>/dev/null || echo /nonexistent)" \
  "home-cache:$HOME/.cache" \
  "tmp:/tmp"
do
  emit "$(printf '%8s MB  %s' "$(mb "${entry#*:}")" "${entry%%:*}")"
done
emit ""

emit "--- prunable preinstalled tooling (safe to delete for Fleet e2e) ---"
prunable_total=0
for path in "${PRUNE_CANDIDATES[@]}"; do
  size=$(mb "$path")
  [ "$size" -gt 0 ] || continue
  prunable_total=$((prunable_total + size))
  emit "$(printf '%8s MB  %s' "$size" "$path")"
done
emit "$(printf '%8s MB  TOTAL RECLAIMABLE' "$prunable_total")"
emit ""

if command -v docker >/dev/null; then
  emit "--- docker ---"
  docker system df 2>/dev/null | tee -a "$report"
  emit ""
  emit "largest images:"
  docker images --format '{{.Size}}\t{{.Repository}}:{{.Tag}}' 2>/dev/null \
    | sort -rh | head -15 | tee -a "$report"
  emit ""
  emit "k3d node containers (writable layer / total):"
  docker ps -s --filter "name=k3d-" \
    --format '{{.Names}}\t{{.Size}}' 2>/dev/null | tee -a "$report"
  emit ""
fi

if command -v kubectl >/dev/null; then
  emit "--- kubelet view (all k3d clusters) ---"
  for ctx in $(kubectl config get-contexts -o name 2>/dev/null | grep '^k3d-'); do
    emit "context: $ctx"
    kubectl --context "$ctx" get nodes \
      -o custom-columns=\
NODE:.metadata.name,\
EPHEMERAL:.status.allocatable.ephemeral-storage,\
DISK_PRESSURE:'.status.conditions[?(@.type=="DiskPressure")].status',\
TAINTS:'.spec.taints[*].key' \
      2>/dev/null | tee -a "$report"
  done
  emit ""
  emit "recent eviction events:"
  for ctx in $(kubectl config get-contexts -o name 2>/dev/null | grep '^k3d-'); do
    kubectl --context "$ctx" get events -A --field-selector reason=Evicted \
      -o custom-columns=NS:.metadata.namespace,OBJ:.involvedObject.name,MSG:.message \
      2>/dev/null | grep -v '^NS ' | sed "s|^|$ctx |" | tee -a "$report"
  done
  emit ""
fi

if [ "$deep" = "--deep" ]; then
  emit "--- top-level consumers of / ---"
  sudo du -xm --max-depth=1 / 2>/dev/null | sort -rn | head -20 | tee -a "$report"
  emit ""
fi

echo "::endgroup::"
