package singlecluster_test

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/rancher/fleet/e2e/testenv"
	"github.com/rancher/fleet/internal/cmd/cli/dump"
)

var _ = Describe("Fleet dump", Label("sharding"), func() {
	When("the cluster has Fleet installed with metrics enabled", func() {
		var (
			testName = "test-cli-dump"
		)

		It("includes metrics into the archive", func() {
			k := env.Kubectl.Namespace(env.Namespace)

			// Create a GitRepo to ensure fleet metrics are populated
			err := testenv.CreateGitRepo(k, env.Namespace, testName, "master", "", "simple")
			Expect(err).ToNot(HaveOccurred())

			// Clean up the test GitRepo after the test
			DeferCleanup(func() {
				out, err := k.Delete("gitrepo", testName)
				Expect(err).ToNot(HaveOccurred(), out)
			})

			// Wait for Bundle to be created and have its status updated
			// This ensures metrics are collected by the Bundle controller
			Eventually(func() bool {
				out, err := k.Namespace(env.Namespace).Get("bundles")
				if err != nil {
					return false
				}
				// Check if at least one bundle exists and has been processed
				return strings.Contains(out, testName)
			}).Should(BeTrue())

			tgzPath := "test.tgz"

			err = dump.Create(context.Background(), restConfig, tgzPath, dump.Options{})
			Expect(err).ToNot(HaveOccurred())

			defer func() {
				Expect(os.RemoveAll(tgzPath)).ToNot(HaveOccurred())
			}()

			f, err := os.OpenFile(tgzPath, os.O_RDONLY, 0)
			Expect(err).ToNot(HaveOccurred())

			defer f.Close()

			gzr, err := gzip.NewReader(f)
			Expect(err).ToNot(HaveOccurred())

			tr := tar.NewReader(gzr)

			foundFiles := []string{}
			for {
				header, err := tr.Next()
				if errors.Is(err, io.EOF) {
					break
				}

				Expect(err).ToNot(HaveOccurred())
				Expect(int32(header.Typeflag)).To(Equal(tar.TypeReg)) // regular file

				content, err := io.ReadAll(tr)
				Expect(err).ToNot(HaveOccurred())

				fileName := strings.Split(header.Name, "_")

				kindLow := fileName[0]
				if kindLow != "metrics" {
					continue
				}

				Expect(fileName).To(HaveLen(2))
				Expect(content).ToNot(BeEmpty())

				// Run a few basic checks on expected strings, checking full contents would be cumbersome
				c := string(content)
				Expect(c).To(ContainSubstring("controller_runtime_active_workers"))
				Expect(c).To(ContainSubstring("controller_runtime_max_concurrent_reconciles"))
				Expect(c).To(ContainSubstring("controller_runtime_reconcile_total"))

				exampleMonitoredRsc := "bundle"
				if strings.Contains(fileName[1], "gitjob") {
					exampleMonitoredRsc = "gitrepo"
				} else if !strings.Contains(fileName[1], "shard") {
					// Check for fleet_*_desired_ready metrics on non-sharded services
					Expect(c).To(ContainSubstring(fmt.Sprintf("fleet_%s_desired_ready", exampleMonitoredRsc)))
				}

				Expect(c).To(ContainSubstring(fmt.Sprintf(`workqueue_work_duration_seconds_bucket{controller="%s",name="%s",`, exampleMonitoredRsc, exampleMonitoredRsc)))

				foundFiles = append(foundFiles, header.Name)
			}

			Expect(foundFiles).To(HaveLen(8))
			Expect(foundFiles).To(ContainElement("metrics_monitoring-gitjob"))
			Expect(foundFiles).To(ContainElement("metrics_monitoring-gitjob-shard-shard0"))
			Expect(foundFiles).To(ContainElement("metrics_monitoring-gitjob-shard-shard1"))
			Expect(foundFiles).To(ContainElement("metrics_monitoring-gitjob-shard-shard2"))
			Expect(foundFiles).To(ContainElement("metrics_monitoring-fleet-controller"))
			Expect(foundFiles).To(ContainElement("metrics_monitoring-fleet-controller-shard-shard0"))
			Expect(foundFiles).To(ContainElement("metrics_monitoring-fleet-controller-shard-shard1"))
			Expect(foundFiles).To(ContainElement("metrics_monitoring-fleet-controller-shard-shard2"))
		})
	})

	When("filtering by namespace", func() {
		var (
			testName      = "test-cli-dump-target-namespace"
			otherTestName = "test-cli-dump-other-namespace"
			targetNs      = "fleet-local" // We need BundleDeployments created, hence use fleet-local
			otherNs       = "fleet-other"
			tgzPath       = "test-namespace-filter.tgz"
			tests         = []struct {
				name      string
				namespace string
			}{
				{
					name:      testName,
					namespace: targetNs,
				},
				{
					name:      otherTestName,
					namespace: otherNs,
				},
			}
		)

		BeforeEach(func() {
			k := env.Kubectl

			for _, test := range tests {
				// Create namespace
				if test.namespace != "fleet-local" {
					out, err := k.Create("namespace", test.namespace)
					Expect(err).ToNot(HaveOccurred(), out)
				}

				// Create GitRepo in target namespace
				err := testenv.CreateGitRepo(k.Namespace(test.namespace), test.namespace, test.name, "master", "", "simple")
				Expect(err).ToNot(HaveOccurred())

				// Wait for bundles to be created in both namespaces
				Eventually(func() bool {
					out, err := k.Namespace(test.namespace).Get("bundles")
					if err != nil {
						return false
					}
					return strings.Contains(out, test.name)
				}, 30*time.Second, 2*time.Second).Should(BeTrue())
			}
		})

		AfterEach(func() {
			k := env.Kubectl

			_, _ = k.Delete("namespace", otherNs) // just delete the extra namespace (inclusive GitRepo)
			_, _ = k.Delete("gitrepo", testName)
			_ = os.RemoveAll(tgzPath)
		})

		It("dumps only resources from the specified namespace", func() {
			// Create dump filtered by target namespace
			err := dump.Create(context.Background(), restConfig, tgzPath, dump.Options{
				Namespace:     targetNs,
				AllNamespaces: false,
			})
			Expect(err).ToNot(HaveOccurred())

			// Parse the archive and collect dumped resources
			dumpedResources := extractResourcesFromArchive(tgzPath)

			// Verify GitRepos
			Expect(dumpedResources["gitrepos"]).To(ContainElement(ContainSubstring(testName)),
				"Should include GitRepo from target namespace")
			Expect(dumpedResources["gitrepos"]).ToNot(ContainElement(ContainSubstring(otherTestName)),
				"Should NOT include GitRepo from other namespace")

			// Verify Bundles
			Expect(dumpedResources["bundles"]).To(ContainElement(ContainSubstring(testName)),
				"Should include Bundles from target namespace")
			Expect(dumpedResources["bundles"]).ToNot(ContainElement(ContainSubstring(otherTestName)),
				"Should NOT include Bundles from other namespace")

			// Verify BundleDeployments are included (they're in cluster namespace but labeled with bundle-namespace)
			foundBundleDeployments := false
			for _, bd := range dumpedResources["bundledeployments"] {
				if strings.Contains(bd, testName) {
					foundBundleDeployments = true
					break
				}
			}
			Expect(foundBundleDeployments).To(BeTrue(),
				"Should include BundleDeployments related to bundles in target namespace")

			// Verify we don't have BundleDeployments from other namespace
			foundOtherBundleDeployments := false
			for _, bd := range dumpedResources["bundledeployments"] {
				if strings.Contains(bd, otherTestName) {
					foundOtherBundleDeployments = true
					break
				}
			}
			Expect(foundOtherBundleDeployments).To(BeFalse(),
				"Should NOT include BundleDeployments from other namespace")
		})

		It("dumps all resources when using --all-namespaces", func() {
			// Create dump with all-namespaces flag
			err := dump.Create(context.Background(), restConfig, tgzPath, dump.Options{
				AllNamespaces: true,
			})
			Expect(err).ToNot(HaveOccurred())

			// Parse the archive and collect dumped resources
			dumpedResources := extractResourcesFromArchive(tgzPath)

			// Verify both GitRepos are included
			Expect(dumpedResources["gitrepos"]).To(ContainElement(ContainSubstring(testName)),
				"Should include GitRepo from target namespace")
			Expect(dumpedResources["gitrepos"]).To(ContainElement(ContainSubstring(otherTestName)),
				"Should include GitRepo from other namespace")

			// Verify both Bundles are included
			Expect(dumpedResources["bundles"]).To(ContainElement(ContainSubstring(testName)),
				"Should include Bundles from target namespace")
			Expect(dumpedResources["bundles"]).To(ContainElement(ContainSubstring(otherTestName)),
				"Should include Bundles from other namespace")
		})
	})
})

// extractResourcesFromArchive extracts resources from a dump archive and returns a map of resource types to file names
func extractResourcesFromArchive(archivePath string) map[string][]string {
	resources := make(map[string][]string)

	f, err := os.OpenFile(archivePath, os.O_RDONLY, 0)
	Expect(err).ToNot(HaveOccurred())
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	Expect(err).ToNot(HaveOccurred())

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		Expect(err).ToNot(HaveOccurred())

		// Extract resource type from filename (format: resourcetype_namespace_name)
		parts := strings.Split(header.Name, "_")
		if len(parts) > 0 {
			resourceType := parts[0]
			resources[resourceType] = append(resources[resourceType], header.Name)
		}
	}

	return resources
}
