package target

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/davecgh/go-spew/spew"
	fleet "github.com/rancher/fleet/pkg/apis/fleet.cattle.io/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// createTargets creates a slice of targets with sequentially numbered clusters
// and bundles. Both values, start and stop, are inclusive, meaning the targets
// will be created from start to stop and having that number in the deployment
// id.
func createTargets(start, stop int) []*Target {
	targets := make([]*Target, stop-start+1)
	for i := range stop - start + 1 {
		targets[i] = &Target{
			Cluster: &fleet.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "cluster-" + strconv.Itoa(start),
				},
			},
			Bundle: &fleet.Bundle{
				ObjectMeta: metav1.ObjectMeta{
					Name: "bundle-" + strconv.Itoa(start),
				},
			},
			Deployment: &fleet.BundleDeployment{
				ObjectMeta: metav1.ObjectMeta{},
				Spec:       fleet.BundleDeploymentSpec{},
				Status:     fleet.BundleDeploymentStatus{},
			},
			DeploymentID: "deployment-" + strconv.Itoa(start),
		}
		start++
	}
	return targets
}

func Test_createTargets(t *testing.T) {
	tests := []struct {
		name        string
		start, stop int
		want        []*Target
	}{
		{
			name:  "start and stop should be inclusive",
			start: 1,
			stop:  5,
			want: []*Target{
				{DeploymentID: "deployment-1"},
				{DeploymentID: "deployment-2"},
				{DeploymentID: "deployment-3"},
				{DeploymentID: "deployment-4"},
				{DeploymentID: "deployment-5"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := createTargets(tt.start, tt.stop)
			if err := targetsEqual(got, tt.want); err != nil {
				t.Errorf("createTargets(%d, %d): %v", tt.start, tt.stop, err)
			}
		})
	}
}

func withCluster(targets []*Target, cluster *fleet.Cluster) {
	for _, target := range targets {
		target.Cluster = cluster
	}
}

func withClusterGroup(targets []*Target, clusterGroup *fleet.ClusterGroup) {
	for _, target := range targets {
		target.ClusterGroups = append(target.ClusterGroups, clusterGroup)
	}
}

func targetsEqual(got, want []*Target) error {
	if len(want) != len(got) {
		return fmt.Errorf("targets have different lengths: got %d but want %d", len(got), len(want))
	}

	for i := range want {
		if want[i].DeploymentID != got[i].DeploymentID {
			return fmt.Errorf("target %d has different deployment IDs: got %v but want %v", i, got[i], want[i])
		}
	}

	return nil
}

// partitionsEqual compares two slices of partitions for equality. It ignores
// the status of the partitions.
func partitionsEqual(got, want []partition) error {
	type targetStats struct {
		length   int
		from, to string
	}
	stats := func() []*targetStats {
		stats := make([]*targetStats, len(got))
		for i, p := range got {
			stats[i] = &targetStats{
				length: len(p.Targets),
				// targets are sorted by Name
				from: p.Targets[0].DeploymentID,
				to:   p.Targets[len(p.Targets)-1].DeploymentID,
			}
		}
		return stats
	}
	if len(got) != len(want) {
		return fmt.Errorf("partitions have different lengths: got %d but want %d\nstats of got: %s", len(got), len(want), spew.Sdump(stats()))
	}
	for i := range want {
		if err := targetsEqual(got[i].Targets, want[i].Targets); err != nil {
			return fmt.Errorf("partition %d has different targets: %v\nstats of got: %s", i, err, spew.Sdump(stats()))
		}

		if got[i].Status.MaxUnavailable != want[i].Status.MaxUnavailable {
			return fmt.Errorf(
				"partition %d has different MaxUnavailable: got %d but want %d",
				i,
				got[i].Status.MaxUnavailable,
				want[i].Status.MaxUnavailable,
			)
		}
	}
	return nil
}

func Test_autoPartition(t *testing.T) {
	tests := []struct {
		name    string
		rollout *fleet.RolloutStrategy
		targets []*Target
		want    []partition
		wantErr bool
	}{
		{
			name: "should partition according to fixed AutoPartitionSize (only >=200 clusters)",
			rollout: &fleet.RolloutStrategy{
				AutoPartitionSize: &intstr.IntOrString{Type: intstr.Int, IntVal: 100},
			},
			targets: createTargets(1, 200),
			want: []partition{
				{Targets: createTargets(1, 100), Status: fleet.PartitionStatus{MaxUnavailable: 100}},
				{Targets: createTargets(101, 200), Status: fleet.PartitionStatus{MaxUnavailable: 100}},
			},
		},
		{
			name: "less than 200 targets should all be in one partition",
			rollout: &fleet.RolloutStrategy{
				AutoPartitionSize: &intstr.IntOrString{Type: intstr.String, StrVal: "25%"},
				MaxUnavailable:    &intstr.IntOrString{Type: intstr.Int, IntVal: 1},
			},
			targets: createTargets(1, 199),
			want: []partition{
				{Targets: createTargets(1, 199), Status: fleet.PartitionStatus{MaxUnavailable: 1}},
			},
		},
		{
			name:    "with 200 targets and above, we expect 4 partitions with a default of 25% for partition size",
			rollout: &fleet.RolloutStrategy{}, // MaxUnavailable defaults to 100%
			targets: createTargets(1, 200),
			want: []partition{
				{Targets: createTargets(1, 50), Status: fleet.PartitionStatus{MaxUnavailable: 50}},
				{Targets: createTargets(51, 100), Status: fleet.PartitionStatus{MaxUnavailable: 50}},
				{Targets: createTargets(101, 150), Status: fleet.PartitionStatus{MaxUnavailable: 50}},
				{Targets: createTargets(151, 200), Status: fleet.PartitionStatus{MaxUnavailable: 50}},
			},
		},
		{
			name: "rest ends up in a separate partition",
			rollout: &fleet.RolloutStrategy{
				AutoPartitionSize: &intstr.IntOrString{Type: intstr.String, StrVal: "49%"},
			},
			targets: createTargets(1, 1000),
			want: []partition{
				{Targets: createTargets(1, 490), Status: fleet.PartitionStatus{MaxUnavailable: 490}},
				{Targets: createTargets(491, 980), Status: fleet.PartitionStatus{MaxUnavailable: 490}},
				{Targets: createTargets(981, 1000), Status: fleet.PartitionStatus{MaxUnavailable: 20}},
			},
		},
		{
			name: "MaxUnavailable from RolloutStrategy should be used in each partition",
			rollout: &fleet.RolloutStrategy{
				AutoPartitionSize: &intstr.IntOrString{Type: intstr.String, StrVal: "10%"},
				MaxUnavailable:    &intstr.IntOrString{Type: intstr.String, StrVal: "10%"},
			},
			targets: createTargets(1, 1000),
			want: []partition{
				{Targets: createTargets(1, 100), Status: fleet.PartitionStatus{MaxUnavailable: 10}},
				{Targets: createTargets(101, 200), Status: fleet.PartitionStatus{MaxUnavailable: 10}},
				{Targets: createTargets(201, 300), Status: fleet.PartitionStatus{MaxUnavailable: 10}},
				{Targets: createTargets(301, 400), Status: fleet.PartitionStatus{MaxUnavailable: 10}},
				{Targets: createTargets(401, 500), Status: fleet.PartitionStatus{MaxUnavailable: 10}},
				{Targets: createTargets(501, 600), Status: fleet.PartitionStatus{MaxUnavailable: 10}},
				{Targets: createTargets(601, 700), Status: fleet.PartitionStatus{MaxUnavailable: 10}},
				{Targets: createTargets(701, 800), Status: fleet.PartitionStatus{MaxUnavailable: 10}},
				{Targets: createTargets(801, 900), Status: fleet.PartitionStatus{MaxUnavailable: 10}},
				{Targets: createTargets(901, 1000), Status: fleet.PartitionStatus{MaxUnavailable: 10}},
			},
		},
		{
			name:    "rounding with 230 clusters",
			rollout: &fleet.RolloutStrategy{},
			targets: createTargets(1, 230),
			want: []partition{
				{Targets: createTargets(1, 57), Status: fleet.PartitionStatus{MaxUnavailable: 57}},
				{Targets: createTargets(58, 114), Status: fleet.PartitionStatus{MaxUnavailable: 57}},
				{Targets: createTargets(115, 171), Status: fleet.PartitionStatus{MaxUnavailable: 57}},
				{Targets: createTargets(172, 228), Status: fleet.PartitionStatus{MaxUnavailable: 57}},
				{Targets: createTargets(229, 230), Status: fleet.PartitionStatus{MaxUnavailable: 2}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := autoPartition(tt.rollout, tt.targets)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("autoPartition() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("autoPartition() succeeded unexpectedly")
			}

			if err := partitionsEqual(got, tt.want); err != nil {
				t.Errorf("autoPartition(): %v", err)
			}
		})
	}
}

func Test_manualPartition(t *testing.T) {
	tests := []struct {
		name      string
		rollout   *fleet.RolloutStrategy
		targets   []*Target
		targetsFn func() []*Target
		want      []partition
		wantErr   bool
	}{
		{
			name: "should match cluster names",
			rollout: &fleet.RolloutStrategy{
				Partitions: []fleet.Partition{
					{
						Name:        "Partition 1",
						ClusterName: "cluster-1",
					},
					{
						Name:        "Partition 2",
						ClusterName: "cluster-2",
					},
				},
			},
			targetsFn: func() []*Target {
				cluster1 := &fleet.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster-1",
					},
				}
				cluster2 := &fleet.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster-2",
					},
				}
				targets := createTargets(1, 4)
				withCluster(targets[0:2], cluster1)
				withCluster(targets[2:4], cluster2)
				return targets
			},
			want: []partition{
				{
					Targets: createTargets(1, 2),
					Status:  fleet.PartitionStatus{MaxUnavailable: 2},
				},
				{
					Targets: createTargets(3, 4),
					Status:  fleet.PartitionStatus{MaxUnavailable: 2},
				},
			},
		},
		{
			name: "should match cluster groups",
			rollout: &fleet.RolloutStrategy{
				Partitions: []fleet.Partition{
					{
						Name:         "Partition 1",
						ClusterGroup: "group-1",
					},
					{
						Name:         "Partition 2",
						ClusterGroup: "group-2",
					},
				},
			},
			targetsFn: func() []*Target {
				targets := createTargets(1, 4)
				cluster1 := &fleet.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster-1",
					},
				}
				cluster2 := &fleet.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster-2",
					},
				}
				withCluster(targets[0:2], cluster1)
				withCluster(targets[2:4], cluster2)
				withClusterGroup(targets[0:2], &fleet.ClusterGroup{
					ObjectMeta: metav1.ObjectMeta{
						Name: "group-1",
					},
					Spec: fleet.ClusterGroupSpec{
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"group": "group-1"},
						},
					},
				})
				withClusterGroup(targets[2:4], &fleet.ClusterGroup{
					ObjectMeta: metav1.ObjectMeta{
						Name: "group-2",
					},
					Spec: fleet.ClusterGroupSpec{
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"group": "group-2"},
						},
					},
				})
				return targets
			},
			want: []partition{
				{Targets: createTargets(1, 2), Status: fleet.PartitionStatus{MaxUnavailable: 2}},
				{Targets: createTargets(3, 4), Status: fleet.PartitionStatus{MaxUnavailable: 2}},
			},
		},
		{
			name: "selectors that match more than once should lead to having targets in multiple partitions",
			rollout: &fleet.RolloutStrategy{
				Partitions: []fleet.Partition{
					{ClusterSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"env": "testing"}}},
					{ClusterSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"group": "a"}}},
				},
			},
			targetsFn: func() []*Target {
				targets := createTargets(1, 100)
				withCluster(targets[0:40], &fleet.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"env": "testing",
						},
					},
				})
				withCluster(targets[40:60], &fleet.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"env":   "testing",
							"group": "a",
						},
					},
				})
				withCluster(targets[60:100], &fleet.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"group": "a",
						},
					},
				})
				return targets
			},
			want: []partition{
				{Targets: createTargets(1, 60), Status: fleet.PartitionStatus{MaxUnavailable: 60}},
				{Targets: createTargets(41, 100), Status: fleet.PartitionStatus{MaxUnavailable: 60}},
			},
		},
		{
			name: "doesn't put unmatched clusters in separate partitions at the end using AutoPartitionSize",
			rollout: &fleet.RolloutStrategy{
				AutoPartitionSize: &intstr.IntOrString{Type: intstr.String, StrVal: "50%"},
				Partitions: []fleet.Partition{
					{Name: "first", ClusterSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"group": "a"}}},
				},
			},
			targetsFn: func() []*Target {
				targets := createTargets(1, 100)
				withCluster(targets[0:50], &fleet.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"group": "a",
						},
					},
				})
				return targets
			},
			want: []partition{
				{Targets: createTargets(1, 50), Status: fleet.PartitionStatus{MaxUnavailable: 50}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var targets []*Target
			if len(tt.targets) > 0 {
				targets = tt.targets
			} else {
				targets = tt.targetsFn()
			}
			got, gotErr := manualPartition(tt.rollout, targets)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("manualPartition() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("manualPartition() succeeded unexpectedly")
			}
			if err := partitionsEqual(got, tt.want); err != nil {
				fmt.Printf("got: %+v\n", got)
				fmt.Printf("want: %+v\n", tt.want)
				t.Errorf("manualPartition(): %v", err)
			}
		})
	}
}
