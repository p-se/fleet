package agent

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/davecgh/go-spew/spew"
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

	createConfigMap := func(data, labels, annotations map[string]string) *corev1.ConfigMap {
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

	// assertConfigMap checks that the ConfigMap exists and that it passes the
	// provided validate function.
	assertConfigMap := func(validate func(corev1.ConfigMap) bool) {
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
		}).Should(BeTrue(), "ConfigMap does not match expected state: %+v", cm)
	}

	// assertBundleDeployment checks that the BundleDeployment exists and that it
	// passes the provided validate function.
	assertBundleDeployment := func(name string, validate func(*v1alpha1.BundleDeployment) error) {
		bd := v1alpha1.BundleDeployment{}
		Eventually(func() error {
			err := k8sClient.Get(
				ctx,
				types.NamespacedName{Namespace: clusterNS, Name: name},
				&bd,
			)
			if err != nil {
				return err
			}
			return validate(&bd)
		}).Should(Succeed(), "BundleDeployment does not match expected state: %+v", bd)
	}

	// mapPartialMatch checks that the super map contains all the keys and values
	// of the sub map. If the value in the sub map is an empty string, the key
	// must exist in the super map but the value is not compared.
	// TODO make it return an error to provide better feedback
	mapPartialMatch := func(super, sub map[string]string) bool {
		for k, v := range sub {
			if v == "" {
				if _, ok := super[k]; !ok {
					return false
				}
			} else {
				if v2, ok := super[k]; !ok || v2 != v {
					return false
				}
			}
		}
		return true
	}

	// configMapAdoptedAndMerged checks that the ConfigMap is adopted. It may
	// need to be extended to check for more labels and annotations.
	isConfigMapAdopted := func(cm *corev1.ConfigMap) bool {
		spew.Dump("==> Labels", cm.Labels)
		spew.Dump("==> Annotations", cm.Annotations)
		return mapPartialMatch(cm.Labels, map[string]string{
			"app.kubernetes.io/managed-by": "Helm",
		}) && mapPartialMatch(cm.Annotations, map[string]string{
			"meta.helm.sh/release-name":      "",
			"meta.helm.sh/release-namespace": "",
		})
	}

	changeConfigMap := func(name string, change func(*corev1.ConfigMap)) {
		cm := &corev1.ConfigMap{}
		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, cm)
		}).Should(Succeed())
		change(cm)
		Eventually(func() error {
			return k8sClient.Update(ctx, cm)
		}).Should(Succeed())
	}

	deleteConfigMap := func(name string) {
		cm := &corev1.ConfigMap{}
		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, cm)
		}).Should(Succeed())
		Eventually(func() error {
			return k8sClient.Delete(ctx, cm)
		}).Should(Succeed())
	}

	bundleDeploymentResourceMissing := func(bd *v1alpha1.BundleDeployment) error {
		for _, condition := range bd.Status.Conditions {
			if condition.Type == v1alpha1.BundleDeploymentConditionReady &&
				condition.Status == corev1.ConditionFalse &&
				condition.Reason == "Error" &&
				strings.Contains(condition.Message, "missing") {
				return nil
			}
		}
		return fmt.Errorf("does not match expected condition")
	}

	bundleDeploymentNotOwnedByUs := func(bd *v1alpha1.BundleDeployment) error {
		for _, condition := range bd.Status.Conditions {
			if condition.Type == v1alpha1.BundleDeploymentConditionReady &&
				condition.Status == corev1.ConditionFalse &&
				condition.Reason == "Error" &&
				strings.Contains(condition.Message, "not owned by us") {
				return nil
			}
		}
		return fmt.Errorf("does not match expected condition")
	}

	bundleDeploymentReady := func(bd *v1alpha1.BundleDeployment) error {
		for _, condition := range bd.Status.Conditions {
			if condition.Type == v1alpha1.BundleDeploymentConditionReady &&
				condition.Status == corev1.ConditionTrue {
				return nil
			}
		}
		return fmt.Errorf("does not match expected condition")
	}

	BeforeEach(func() {
		namespace = createNamespace()
		DeferCleanup(func() {
			Expect(k8sClient.Delete(ctx, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: namespace}})).ToNot(HaveOccurred())
		})
		env = &specEnv{namespace: namespace}
	})

	When("a resource of a bundle deployment is removed", Label("remove-resource"), func() {
		It("should report the deleted resource as missing", func() {
			createBundleDeployment("remove-resource", false)
			assertConfigMap(func(cm corev1.ConfigMap) bool {
				return mapPartialMatch(cm.Data, map[string]string{"key": "value"}) &&
					mapPartialMatch(cm.Annotations, map[string]string{
						"meta.helm.sh/release-name":      "remove-resource",
						"meta.helm.sh/release-namespace": "",
						"objectset.rio.cattle.io/id":     "default-remove-resource",
					}) && mapPartialMatch(cm.Labels, map[string]string{
					"app.kubernetes.io/managed-by": "Helm",
					"objectset.rio.cattle.io/hash": "ca7682543199bb801d0c14587a1158d936508160",
				})
			})
			deleteConfigMap("cm1")
			assertBundleDeployment("remove-resource", bundleDeploymentResourceMissing)
		})
	})

	When("all labels of a resource of a bundle deployment are removed", Label("remove-labels"), func() {
		It("should report that resource as \"not owned by us\"", func() {
			createBundleDeployment("remove-metadata", false)
			assertConfigMap(func(cm corev1.ConfigMap) bool {
				return mapPartialMatch(cm.Data, map[string]string{"key": "value"}) &&
					mapPartialMatch(cm.Annotations, map[string]string{
						"meta.helm.sh/release-name":      "remove-metadata",
						"meta.helm.sh/release-namespace": "",
						"objectset.rio.cattle.io/id":     "default-remove-metadata",
					}) && mapPartialMatch(cm.Labels, map[string]string{
					"app.kubernetes.io/managed-by": "Helm",
					"objectset.rio.cattle.io/hash": "0f3e1d9d146fa8b290c0de403881184751430e59",
				})
			})
			changeConfigMap("cm1", func(cm *corev1.ConfigMap) {
				cm.Labels = map[string]string{}
			})
			assertBundleDeployment("remove-metadata", bundleDeploymentNotOwnedByUs)
		})
	})

	When("a bundle deployment adopts a \"clean\" resource", Label("clean"), func() {
		It("verifies that the ConfigMap is adopted and its content merged", func() {
			createConfigMap(map[string]string{"foo": "bar"}, nil, nil)
			createBundleDeployment("adopt-clean", true)
			assertConfigMap(func(cm corev1.ConfigMap) bool {
				return isConfigMapAdopted(&cm) &&
					mapPartialMatch(cm.Data, map[string]string{"foo": "bar", "key": "value"})
			})
		})
	})

	When("a bundle deployment adopts a resource with wrangler metadata", Label("wrangler-metadata"), func() {
		It("verifies that the ConfigMap is adopted and its content merged", func() {
			createConfigMap(
				map[string]string{"foo": "bar"},
				map[string]string{"objectset.rio.cattle.io/hash": "33ed67317c57ea78702e369c4c025f8df88553cc"},
				map[string]string{"objectset.rio.cattle.io/id": "some-assumed-old-id"},
			)
			createBundleDeployment("adopt-wrangler-metadata", true)
			assertConfigMap(func(cm corev1.ConfigMap) bool {
				return isConfigMapAdopted(&cm) &&
					mapPartialMatch(cm.Data, map[string]string{"foo": "bar", "key": "value"})
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
			assertConfigMap(func(cm corev1.ConfigMap) bool {
				return isConfigMapAdopted(&cm) &&
					mapPartialMatch(cm.Data, map[string]string{"foo": "bar", "key": "value"})
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
			assertConfigMap(func(cm corev1.ConfigMap) bool {
				return isConfigMapAdopted(&cm) &&
					mapPartialMatch(cm.Data, map[string]string{"foo": "bar", "key": "value"})
			})
		})
	})

	When("a bundle adopts a resource that is deployed by another bundle", Label("competing-bundles"), func() {
		It("should complain about not owning the resource", func() {
			createBundleDeployment("one", false)
			waitForConfigMap("cm1")
			createBundleDeployment("two", true)
			assertBundleDeployment("one", bundleDeploymentNotOwnedByUs)
			assertBundleDeployment("two", bundleDeploymentReady)
		})
	})
})
