#!/bin/bash

# Removes preinstalled tooling that Fleet's e2e jobs never touch.
#
# The multi-cluster job runs three k3d clusters as six containers sharing the
# runner's root filesystem. On a private repository that filesystem is 72 GB
# and arrives 82% full, leaving ~13.5 GiB for a job that consumes ~11.5 GiB, so
# it finishes within half a gigabyte of the 5% threshold at which kubelet
# declares DiskPressure, taints the nodes NoSchedule and evicts pods.
#
# Measured contents of these paths on ubuntu24/20260816.277, 25.6 GB in total.
# Removal happens in parallel because the runner has two cores and these are
# large trees.
#
# This script never fails the job.

set -u

PRUNE=(
  /usr/local/lib/android         # 11.07 GB
  /usr/share/dotnet              #  5.66 GB
  /usr/local/.ghcup              #  3.74 GB
  /usr/share/swift               #  3.37 GB
  /opt/hostedtoolcache/CodeQL    #  1.73 GB
)

avail() { df -Pk / | awk 'NR==2 {print $4}'; }
gib() { awk -v k="$1" 'BEGIN {printf "%.2f GiB", k/1048576}'; }

before=$(avail)
echo "::group::Free disk space"
echo "available before: $(gib "$before")"

for path in "${PRUNE[@]}"; do
  [ -e "$path" ] || continue
  echo "removing $path"
  sudo rm -rf "$path" &
done
wait

after=$(avail)
echo "available after:  $(gib "$after")"
echo "reclaimed:        $(gib $((after - before)))"
df -h /
echo "::endgroup::"
