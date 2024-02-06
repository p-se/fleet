package metrics_test

import (
	"fmt"
	"math/rand"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/rancher/fleet/e2e/metrics"
	"github.com/rancher/fleet/e2e/testenv"
	"github.com/rancher/fleet/e2e/testenv/kubectl"
)

var _ = Describe("Bundle Metrics", Label("bundle"), func() {
	const (
		objName = "metrics"
		branch  = "master"
	)

	var (
		// kw is the kubectl command for namespace the workload is deployed to
		kw        kubectl.Command
		namespace string
	)

	BeforeEach(func() {
		k = env.Kubectl.Namespace(env.Namespace)
		namespace = testenv.NewNamespaceName(
			objName,
			rand.New(rand.NewSource(time.Now().UnixNano())),
		)
		kw = k.Namespace(namespace)

		out, err := k.Create("ns", namespace)
		Expect(err).ToNot(HaveOccurred(), out)

		err = testenv.CreateGitRepo(
			kw,
			namespace,
			objName,
			branch,
			"simple-manifest",
		)
		Expect(err).ToNot(HaveOccurred())

		DeferCleanup(func() {
			out, err = k.Delete("ns", namespace)
			Expect(err).ToNot(HaveOccurred(), out)
		})
	})

	When("testing Bundle metrics", func() {
		bundleMetricNames := []string{
			"fleet_bundle_desired_ready",
			"fleet_bundle_err_applied",
			"fleet_bundle_modified",
			"fleet_bundle_not_ready",
			"fleet_bundle_out_of_sync",
			"fleet_bundle_pending",
			"fleet_bundle_ready",
			"fleet_bundle_wait_applied",
		}

		It("should have exactly one metric for the bundle", func() {
			et := metrics.NewExporterTest(metricsURL)
			Eventually(func() error {
				for _, metricName := range bundleMetricNames {
					metric, err := et.FindOneMetric(
						metricName,
						map[string]string{
							"name":      objName + "-simple-manifest",
							"namespace": namespace,
						},
					)
					if err != nil {
						return err
					}
					Expect(metric.Gauge.GetValue()).To(Equal(float64(0)))
				}
				return nil
			}).ShouldNot(HaveOccurred())
		})

		Context("when the GitRepo (and therefore Bundle) is changed", Label("bundle-altered"), func() {
			It("it should not duplicate metrics if Bundle is updated", Label("bundle-update"), func() {
				et := metrics.NewExporterTest(metricsURL)
				out, err := kw.Patch(
					"gitrepo", objName,
					"--type=json",
					"-p", `[{"op": "replace", "path": "/spec/paths", "value": ["simple-chart"]}]`,
				)
				Expect(err).ToNot(HaveOccurred(), out)
				Expect(out).To(ContainSubstring("gitrepo.fleet.cattle.io/metrics patched"))

				// Wait for it to be changed and fetched.
				Eventually(func() (string, error) {
					return kw.Get("gitrepo", objName, "-o", "jsonpath={.status.commit}")
				}).ShouldNot(BeEmpty())

				var metric *metrics.Metric
				// Expect still no metrics to be duplicated.
				Eventually(func() error {
					for _, metricName := range bundleMetricNames {
						metric, err = et.FindOneMetric(
							metricName,
							map[string]string{
								"name":      objName + "-simple-chart",
								"namespace": namespace,
							},
						)
						if err != nil {
							return err
						}
						if metric.LabelValue("paths") == "simple-manifest" {
							return fmt.Errorf("path for metric %s unchanged", metricName)
						}
					}
					return nil
				}).ShouldNot(HaveOccurred())
			})

			It("should not keep metrics if Bundle is deleted", Label("bundle-delete"), func() {
				et := metrics.NewExporterTest(metricsURL)

				objName := objName + "-simple-manifest"

				var (
					out string
					err error
				)
				Eventually(func() error {
					out, err = kw.Delete("bundle", objName)
					return err
				}).ShouldNot(HaveOccurred(), out)

				Eventually(func() error {
					for _, metricName := range bundleMetricNames {
						_, err := et.FindOneMetric(
							metricName,
							map[string]string{
								"name":      objName,
								"namespace": namespace,
							},
						)
						if err == nil {
							return fmt.Errorf("metric %s found but not expected", metricName)
						}
					}
					return nil
				}).ShouldNot(HaveOccurred())
			})
		})
	})
})
