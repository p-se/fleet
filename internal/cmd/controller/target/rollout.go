package target

import (
	"fmt"

	"github.com/rancher/fleet/internal/cmd/controller/target/matcher"
	fleet "github.com/rancher/fleet/pkg/apis/fleet.cattle.io/v1alpha1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

var info = log.Log.V(0).WithName("rollout.go").Info

func partitionInfo(msg string, partitions []partition) {
	info(msg)
	for i, partition := range partitions {
		info(fmt.Sprintf("partition %d", i),
			"status", partition.Status,
			"partitions", len(partition.Targets),
		)
	}
}

// partitions distributes targets into partitions based on the rollout strategy (pure function)
func partitions(targets []*Target) ([]partition, error) {
	rollout := getRollout(targets)
	if len(rollout.Partitions) == 0 {
		partitions, err := autoPartition(rollout, targets)
		partitionInfo("auto partitioning", partitions)
		return partitions, err
	}

	partitions, err := manualPartition(rollout, targets)
	partitionInfo("manual partitioning", partitions)

	return partitions, err
}

// getRollout returns the rollout strategy for the specified targets (pure
// function).
//
// If targets contains several elements, the rollout strategy of the first
// element is used. If no rollout strategy is found, an empty is created and
// returned. This function therefore assumes that all bundles in targets have
// the same rollout strategy.
func getRollout(targets []*Target) *fleet.RolloutStrategy {
	var rollout *fleet.RolloutStrategy
	if len(targets) > 0 {
		rollout = targets[0].Bundle.Spec.RolloutStrategy
	}
	if rollout == nil {
		rollout = &fleet.RolloutStrategy{}
	}
	return rollout
}

// manualPartition computes a slice of Partition given some targets and rollout strategy that already has partitions (pure function)
func manualPartition(rollout *fleet.RolloutStrategy, targets []*Target) ([]partition, error) {
	var (
		partitions []partition
	)

	for _, partitionDef := range rollout.Partitions {
		matcher, err := matcher.NewClusterMatcher(partitionDef.ClusterName, partitionDef.ClusterGroup, partitionDef.ClusterGroupSelector, partitionDef.ClusterSelector)
		if err != nil {
			return nil, err
		}

		var partitionTargets []*Target
	targetLoop:
		for _, target := range targets {
			for _, cg := range target.ClusterGroups {
				if matcher.Match(target.Cluster.Name, cg.Name, cg.Labels, target.Cluster.Labels) {
					partitionTargets = append(partitionTargets, target)
					continue targetLoop
				}
			}
		}

		partitions, err = appendPartition(partitions, partitionDef.Name, partitionTargets, partitionDef.MaxUnavailable, rollout.MaxUnavailable)
		if err != nil {
			return nil, err
		}
	}

	return partitions, nil
}

// autoPartition computes a slice of Partition given some targets and rollout strategy (pure function)
func autoPartition(rollout *fleet.RolloutStrategy, targets []*Target) ([]partition, error) {
	// if auto is disabled
	if rollout.AutoPartitionSize != nil && rollout.AutoPartitionSize.Type == intstr.Int &&
		rollout.AutoPartitionSize.IntVal <= 0 {
		info("disabling auto partitioning, all targets will be in one partition")
		return appendPartition(nil, "All", targets, rollout.MaxUnavailable)
	}

	// Also disable if less than 200
	if len(targets) < 200 {
		info("auto partitioning disabled, less than 200 targets")
		return appendPartition(nil, "All", targets, rollout.MaxUnavailable)
	}

	maxSize, err := limit(len(targets), rollout.AutoPartitionSize, &defAutoPartitionSize)
	if err != nil {
		return nil, err
	}

	var (
		partitions []partition
		offset     = 0
	)

	for {
		if len(targets) == 0 {
			return partitions, nil
		}
		end := min(len(targets), maxSize)

		partitionTargets := targets[:end]
		name := fmt.Sprintf("Partition %d - %d", offset, offset+end)

		partitions, err = appendPartition(partitions, name, partitionTargets, rollout.MaxUnavailable)
		if err != nil {
			return nil, err
		}

		// setup next loop
		targets = targets[end:]
		offset += end
	}
}

// appendPartition appends a new partition to partitions with partitionTargets as targets (does not mutate partitionTargets)
func appendPartition(partitions []partition, name string, partitionTargets []*Target, maxUnavailable ...*intstr.IntOrString) ([]partition, error) {
	maxUnavailableValue, err := limit(len(partitionTargets), maxUnavailable...)
	if err != nil {
		return nil, err
	}
	return append(partitions, partition{
		Status: fleet.PartitionStatus{
			Name:           name,
			Count:          len(partitionTargets),
			MaxUnavailable: maxUnavailableValue,
			Unavailable:    Unavailable(partitionTargets),
			Summary:        Summary(partitionTargets),
		},
		Targets: partitionTargets,
	}), nil
}
