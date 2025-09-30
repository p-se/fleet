package singlecluster_test

// These test cases rely on a local git server, so that they can be run locally and against PRs.
// For tests monitoring external git hosting providers, see `e2e/require-secrets`.

import (
	"fmt"
	"math/rand"
	"os"
	"path"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/rancher/fleet/e2e/testenv"
	"github.com/rancher/fleet/e2e/testenv/githelper"
	"github.com/rancher/fleet/e2e/testenv/kubectl"
)

var _ = FDescribe("Testing go-getter", Label("infra-setup"), func() {
	const (
		sleeper    = "sleeper"
		entrypoint = "entrypoint"
	)

	var (
		tmpDir          string
		cloneDir        string
		k               kubectl.Command
		gh              *githelper.Git
		gitrepoName     string
		r               = rand.New(rand.NewSource(GinkgoRandomSeed()))
		targetNamespace string
		// Build git repo URL reachable _within_ the cluster, for the GitRepo
		host = githelper.BuildGitHostname()
	)

	getExternalRepoURL := func(repoName string) string {
		GinkgoHelper()
		addr, err := githelper.GetExternalRepoAddr(env, HTTPSPort, repoName)
		Expect(err).ToNot(HaveOccurred())
		addr = strings.Replace(addr, "http://", fmt.Sprintf("%s://", "https"), 1)
		return addr
	}

	BeforeEach(func() {
		k = env.Kubectl.Namespace(env.Namespace)
	})

	JustBeforeEach(func() {
		// Create the first repository
		addr := getExternalRepoURL("repo")
		gh = githelper.NewHTTP(addr)
		// sleeperInClusterAddr := gh.GetInClusterURL(host, HTTPSPort, sleeperClusterName)
		tmpDir, err := os.MkdirTemp("", "fleet-")
		Expect(err).ToNot(HaveOccurred())
		cloneDir = path.Join(tmpDir, "repo") // Fixed and built into the container image.
		gitrepoName = testenv.RandomFilename("gitjob-test", r)
		// Creates the content in the sleeperClusterName directory
		_, err = gh.Create(cloneDir, testenv.AssetPath("gitrepo/sleeper-chart"), sleeper)
		Expect(err).ToNot(HaveOccurred())

		// Create the second repository
		Expect(err).ToNot(HaveOccurred())
		tmpAssetDir := path.Join(tmpDir, "entryPoint")
		err = os.Mkdir(tmpAssetDir, 0755)
		Expect(err).ToNot(HaveOccurred())
		url := "git::" + gh.GetInClusterURL(host, HTTPSPort, "repo?ref="+sleeper)
		err = os.WriteFile(
			path.Join(tmpAssetDir, "fleet.yaml"),
			fmt.Appendf([]byte{}, "helm:\n  chart: %s\n", url),
			0755,
		)
		Expect(err).NotTo(HaveOccurred())

		_, err = gh.Create(cloneDir, tmpAssetDir, entrypoint)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		_ = os.RemoveAll(tmpDir)
		_, _ = k.Delete("gitrepo", gitrepoName)

		// Check that the bundle deployment resource has been deleted
		Eventually(func(g Gomega) {
			out, _ := k.Get(
				"bundledeployments",
				"-A",
				"-l",
				fmt.Sprintf("fleet.cattle.io/repo-name=%s", gitrepoName),
			)
			g.Expect(out).To(ContainSubstring("No resources found"))
		}).Should(Succeed())

		// TODO: uncomment
		// _, err := k.Delete("ns", targetNamespace, "--wait=true")
		// Expect(err).ToNot(HaveOccurred())
	})

	When("testing InsecureSkipTLSVerify", func() {
		BeforeEach(func() {
			targetNamespace = testenv.NewNamespaceName("target", r)
		})

		It("should fail if InsecureSkipTLSVerify is false", func() {
			// Create and apply GitRepo
			err := testenv.ApplyTemplate(k, testenv.AssetPath("gitrepo/gitrepo.yaml"), struct {
				Name                  string
				Repo                  string
				Branch                string
				PollingInterval       string
				TargetNamespace       string
				Path                  string
				InsecureSkipTLSVerify string
			}{
				gitrepoName,
				gh.GetInClusterURL(host, HTTPSPort, "repo"),
				gh.Branch,
				"15s",           // default
				targetNamespace, // to avoid conflicts with other tests
				entrypoint,
				"false",
			})
			Expect(err).ToNot(HaveOccurred())

			Eventually(func(g Gomega) {
				out, err := k.Get("gitrepo", gitrepoName, `-o=jsonpath={.status.conditions[?(@.type=="Stalled")].message}`)
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(out).To(ContainSubstring("SSL certificate problem: unable to get local issuer certificate"))
			}).Should(Succeed())
		})

		It("should succeed if InsecureSkipTLSVerify is true", func() {
			// Create and apply GitRepo
			err := testenv.ApplyTemplate(k, testenv.AssetPath("gitrepo/gitrepo.yaml"), struct {
				Name                  string
				Repo                  string
				Branch                string
				PollingInterval       string
				TargetNamespace       string
				Path                  string
				InsecureSkipTLSVerify string
			}{
				gitrepoName,
				gh.GetInClusterURL(host, HTTPSPort, "repo"),
				gh.Branch,
				"15s",           // default
				targetNamespace, // to avoid conflicts with other tests
				entrypoint,
				"true",
			})
			Expect(err).ToNot(HaveOccurred())

			Eventually(func(g Gomega) {
				out, err := k.Get("gitrepo", gitrepoName, `-o=jsonpath={.status.display.readyBundleDeployments}`)
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(out).To(ContainSubstring("1/1"))
			}).Should(Succeed())
		})
	})
})
