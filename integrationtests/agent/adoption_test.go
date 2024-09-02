package agent

import (
	"context"
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/rancher/fleet/pkg/apis/fleet.cattle.io/v1alpha1"
)

func init() {
	configMap, err := os.ReadFile(assetsPath + "/configmap.yaml")
	if err != nil {
		panic(err)
	}

	resources["BundleDeploymentConfigMap"] = []v1alpha1.BundleResource{
		{
			Name:     "configmap.yaml",
			Content:  string(configMap),
			Encoding: "",
		},
	}
}

var _ = Describe("Adoption", Ordered, Label("adopt"), func() {
	var (
		namespace string
		env       *specEnv
	)

	createBundleDeployment := func(name string, takeOwnership bool) {
		bundled := v1alpha1.BundleDeployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: clusterNS,
			},
			Spec: v1alpha1.BundleDeploymentSpec{
				DeploymentID: "BundleDeploymentConfigMap",
				Options: v1alpha1.BundleDeploymentOptions{
					DefaultNamespace: namespace,
					Helm: &v1alpha1.HelmOptions{
						TakeOwnership: takeOwnership,
					},
				},
			},
		}

		err := k8sClient.Create(context.TODO(), &bundled)
		Expect(err).To(BeNil())
		Expect(bundled).To(Not(BeNil()))
		Expect(bundled.Spec.DeploymentID).ToNot(Equal(bundled.Status.AppliedDeploymentID))
		Expect(bundled.Status.Ready).To(BeFalse())
		Eventually(func() bool {
			err := k8sClient.Get(context.TODO(), types.NamespacedName{Namespace: clusterNS, Name: name}, &bundled)
			if err != nil {
				return false
			}
			return bundled.Status.Ready
		}).Should(BeTrue(), "BundleDeployment not ready: status: %+v", bundled.Status)
		Expect(bundled.Spec.DeploymentID).To(Equal(bundled.Status.AppliedDeploymentID))
	}

	waitForConfigMap := func(name string) {
		Eventually(func() error {
			_, err := env.getConfigMap(name)
			return err
		}).Should(Succeed())
	}

	createConfigMap := func(
		data map[string]string,
		labels map[string]string,
		annotations map[string]string,
	) *corev1.ConfigMap {
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "cm1",
				Namespace:   namespace,
				Labels:      labels,
				Annotations: annotations,
			},
			Data: data,
		}
		Expect(k8sClient.Create(ctx, cm)).To(Succeed())
		waitForConfigMap("cm1")
		return cm
	}

	configMapValidates := func(validate func(corev1.ConfigMap) bool) {
		cm := corev1.ConfigMap{}
		Eventually(func() bool {
			err := k8sClient.Get(
				ctx,
				types.NamespacedName{Namespace: namespace, Name: "cm1"},
				&cm,
			)
			if err != nil {
				return false
			}
			return validate(cm)
		}).Should(BeTrue())
	}

	// configMapAdoptedAndMerged checks that the ConfigMap is adopted. It may
	// need to be extended to check for more labels and annotations.
	configMapIsAdopted := func(cm *corev1.ConfigMap) bool {
		expectedAnnotations := []string{
			"meta.helm.sh/release-name",
			"meta.helm.sh/release-namespace",
		}
		for _, annotation := range expectedAnnotations {
			if _, ok := cm.Annotations[annotation]; !ok {
				return false
			}
		}

		if v, ok := cm.Labels["app.kubernetes.io/managed-by"]; !ok || v != "Helm" {
			return false
		}
		return true
	}

	configMapDataEquals := func(cm *corev1.ConfigMap, data map[string]string) bool {
		for k, v := range data {
			if v2, ok := cm.Data[k]; !ok || v2 != v {
				return false
			}
		}
		return true
	}

	BeforeEach(func() {
		namespace = createNamespace()
		DeferCleanup(func() {
			Expect(k8sClient.Delete(ctx, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: namespace}})).ToNot(HaveOccurred())
		})
		env = &specEnv{namespace: namespace}
	})

	When("a bundle deployment adopts a \"clean\" resource", Label("clean"), func() {
		It("verifies that the ConfigMap is adopted and its content merged", func() {
			createConfigMap(map[string]string{"foo": "bar"}, nil, nil)
			createBundleDeployment("adopt-clean", true)
			configMapValidates(func(cm corev1.ConfigMap) bool {
				return configMapIsAdopted(&cm) &&
					configMapDataEquals(&cm, map[string]string{
						"foo": "bar",
						"key": "value",
					})
			})
		})
	})

	When("a bundle deployment adopts a resource with wrangler metadata", Label("wrangler-metadata"), func() {
		It("verifies that the ConfigMap is adopted and its content merged", func() {
			createConfigMap(
				map[string]string{"foo": "bar"},
				map[string]string{
					"objectset.rio.cattle.io/hash": "33ed67317c57ea78702e369c4c025f8df88553cc",
				},
				map[string]string{
					"objectset.rio.cattle.io/id": "some-assumed-old-id",
				},
			)
			createBundleDeployment("adopt-wrangler-metadata", true)
			configMapValidates(func(cm corev1.ConfigMap) bool {
				return configMapIsAdopted(&cm) &&
					configMapDataEquals(&cm, map[string]string{
						"foo": "bar",
						"key": "value",
					})
			})
		})
	})

	When("a bundle deployment adopts a resource with invalid wrangler metadata", Label("wrangler-metadata"), func() {
		It("verifies that the ConfigMap is adopted and its content merged", func() {
			createConfigMap(
				map[string]string{"foo": "bar"},
				map[string]string{"objectset.rio.cattle.io/hash": "234"},
				map[string]string{"objectset.rio.cattle.io/id": "$#@"},
			)
			createBundleDeployment("adopt-invalid-wrangler-metadata", true)
			configMapValidates(func(cm corev1.ConfigMap) bool {
				return configMapIsAdopted(&cm) &&
					configMapDataEquals(&cm, map[string]string{
						"foo": "bar",
						"key": "value",
					})
			})
		})
	})

	When("a bundle deployment adopts a resource with random metadata", Label("random-metadata"), func() {
		It("verifies that the ConfigMap is adopted and its content merged", func() {
			createConfigMap(
				map[string]string{"foo": "bar"},
				map[string]string{"foo": "234"},
				map[string]string{"bar": "xzy"},
			)
			createBundleDeployment("adopt-random-metadata", true)
			configMapValidates(func(cm corev1.ConfigMap) bool {
				return configMapIsAdopted(&cm) &&
					configMapDataEquals(&cm, map[string]string{
						"foo": "bar",
						"key": "value",
					})
			})
		})
	})

	When("a bundle adopts a resource that is deployed by another bundle", Label("competing-bundles"), func() {
		It("should complain about not owning the resource", func() {
			createBundleDeployment("one", false)
			waitForConfigMap("cm1")
			createBundleDeployment("two", true)

			bd1 := &v1alpha1.BundleDeployment{}
			bd2 := &v1alpha1.BundleDeployment{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Namespace: clusterNS, Name: "one"}, bd1)
				if err != nil {
					return false
				}
				err = k8sClient.Get(ctx, types.NamespacedName{Namespace: clusterNS, Name: "two"}, bd2)
				if err != nil {
					return false
				}

				if !bd1.Status.Ready || !bd2.Status.Ready ||
					bd1.Status.NonModified || !bd2.Status.NonModified ||
					strings.Contains(bd1.Status.ModifiedStatus[0].String(), "now owned by us") {
					return false
				}

				return true
			}).Should(BeTrue())
		})
	})
})
