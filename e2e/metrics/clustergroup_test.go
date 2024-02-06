package metrics_test

import (
	"fmt"
	"math/rand"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rancher/fleet/e2e/metrics"
	"github.com/rancher/fleet/e2e/testenv"
)

var _ = Describe("Cluster Metrics", Label("clustergroup"), func() {
	const (
		namespace = "fleet-local"
	)

	var (
		clusterGroupName string
	)

	expectedMetricsExist := map[string]bool{
		"fleet_cluster_group_bundle_desired_ready":         true,
		"fleet_cluster_group_bundle_ready":                 true,
		"fleet_cluster_group_cluster_count":                true,
		"fleet_cluster_group_non_ready_cluster_count":      true,
		"fleet_cluster_group_resource_count_desired_ready": true,
		"fleet_cluster_group_resource_count_missing":       true,
		"fleet_cluster_group_resource_count_modified":      true,
		"fleet_cluster_group_resource_count_notready":      true,
		"fleet_cluster_group_resource_count_orphaned":      true,
		"fleet_cluster_group_resource_count_ready":         true,
		"fleet_cluster_group_resource_count_unknown":       true,
		"fleet_cluster_group_resource_count_waitapplied":   true,
	}

	BeforeEach(func() {
		clusterGroupName = testenv.AddRandomSuffix(
			"test-cluster-group",
			rand.NewSource(time.Now().UnixNano()),
		)
		err := testenv.CreateClusterGroup(
			k,
			namespace,
			clusterGroupName,
			map[string]string{
				"name": "local",
			},
		)
		Expect(err).ToNot(HaveOccurred())

		DeferCleanup(func() {
			out, err := k.Delete(
				"clustergroups.fleet.cattle.io",
				clusterGroupName,
				"-n", namespace,
			)
			Expect(out).To(ContainSubstring("deleted"))
			Expect(err).ToNot(HaveOccurred())
		})
	})

	// The cluster group is created without an UID. This UID is added shortly
	// after the creation of the cluster group. This results in the cluster
	// group being modified and, if not properly checked, duplicated metrics.
	// This is why this test does test for duplicated metrics as well, although
	// it does not look like it.
	It("should have all metrics for a single cluster group once", func() {
		Eventually(func() (string, error) {
			return env.Kubectl.Get(
				"-n", namespace,
				"clustergroups.fleet.cattle.io",
				clusterGroupName,
				"-o", "jsonpath=.metadata.name",
			)
		}).ShouldNot(ContainSubstring("not found"))

		et := metrics.NewExporterTest(metricsURL)

		Eventually(func() error {
			for metricName, expectedExist := range expectedMetricsExist {
				metric, err := et.FindOneMetric(
					metricName,
					map[string]string{
						"name":      clusterGroupName,
						"namespace": namespace,
					},
				)
				if expectedExist {
					if err != nil {
						return err
					}
					Expect(err).ToNot(HaveOccurred())
				} else {
					if err == nil {
						return fmt.Errorf("expected not to exist but found %s", metric)
					}
				}
			}
			return nil
		}).ShouldNot(HaveOccurred())
	},
	)

})
