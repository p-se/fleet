package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/rancher/fleet/pkg/apis/fleet.cattle.io/v1alpha1"
)

func init() {
	resources["BundleDeploymentConfigMap"] = []v1alpha1.BundleResource{
		{
			Name: "configmap.yaml",
			Content: `apiVersion: v1
kind: ConfigMap
metadata:
  name: cm1
data:
  key: value
`,
			Encoding: "",
		},
	}
}

var _ = Describe("Adoption", Label("adopt"), func() {
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
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(
				context.TODO(),
				types.NamespacedName{Namespace: clusterNS, Name: name},
				&bundled,
			)).To(Succeed())
			g.Expect(bundled.Status.Ready).To(BeTrue())
		}).Should(Succeed(), "BundleDeployment not ready: status: %+v", bundled.Status)
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
	assertConfigMap := func(validate func(Gomega, corev1.ConfigMap)) {
		cm := corev1.ConfigMap{}
		var err error
		Eventually(func(g Gomega) {
			err = k8sClient.Get(
				ctx,
				types.NamespacedName{Namespace: namespace, Name: "cm1"},
				&cm,
			)
			g.Expect(err).To(Succeed())
			validate(g, cm)
			g.Expect(err).To(Succeed())
		}).Should(Succeed(), "assertConfigMap error: %v in %+v", err, cm)
	}

	// assertBundleDeployment checks that the BundleDeployment exists and that it
	// passes the provided validate function.
	assertBundleDeployment := func(name string, validate func(*v1alpha1.BundleDeployment) error) {
		bd := v1alpha1.BundleDeployment{}
		var err error
		Eventually(func(g Gomega) {
			err = k8sClient.Get(
				ctx,
				types.NamespacedName{Namespace: clusterNS, Name: name},
				&bd,
			)
			g.Expect(err).ToNot(HaveOccurred())
			err = validate(&bd)
			g.Expect(err).ToNot(HaveOccurred())
		}).Should(Succeed(), "assertBundleDeployment: error %v in %+v", err, bd)
	}

	// configMapAdoptedAndMerged checks that the ConfigMap is adopted. It may
	// need to be extended to check for more labels and annotations.
	assertConfigMapAdopted := func(g Gomega, cm *corev1.ConfigMap) {
		g.Expect(cm.Labels).To(HaveKeyWithValue("app.kubernetes.io/managed-by", "Helm"))
		g.Expect(cm.Annotations).To(HaveKey("meta.helm.sh/release-name"))
		g.Expect(cm.Annotations).To(HaveKey("meta.helm.sh/release-namespace"))
	}

	updateConfigMap := func(name string, update func(*corev1.ConfigMap)) {
		cm := &corev1.ConfigMap{}
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, cm)).To(Succeed())
			update(cm)
			g.Expect(k8sClient.Update(ctx, cm)).To(Succeed())
		}).Should(Succeed())
	}

	deleteConfigMap := func(name string) {
		cm := &corev1.ConfigMap{}
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, cm)).To(Succeed())
			g.Expect(k8sClient.Delete(ctx, cm)).To(Succeed())
		}).Should(Succeed())
	}

	bundleDeploymentResourceMissing := func(bd *v1alpha1.BundleDeployment) error {
		const msgPrefix = "BundleDeployment resource:"
		for _, condition := range bd.Status.Conditions {
			if condition.Type != v1alpha1.BundleDeploymentConditionReady {
				continue
			}
			if condition.Status != corev1.ConditionFalse {
				return fmt.Errorf("%s Status is not False", msgPrefix)
			}
			if condition.Reason != "Error" {
				return fmt.Errorf("%s Reason is not Error", msgPrefix)
			}
			if !strings.Contains(condition.Message, "missing") {
				return fmt.Errorf("%s Message does not contain 'missing'", msgPrefix)
			}
		}
		return nil
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
		env = &specEnv{namespace: namespace}
	})

	When("a resource of a bundle deployment is removed", Label("remove-resource"), func() {
		It("should report the deleted resource as missing", func() {
			createBundleDeployment("remove-resource", false)
			assertConfigMap(func(g Gomega, cm corev1.ConfigMap) {
				g.Expect(cm.Data).To(Equal(map[string]string{"key": "value"}))
				g.Expect(cm.Annotations).To(HaveKeyWithValue("meta.helm.sh/release-name", "remove-resource"))
				g.Expect(cm.Annotations).To(HaveKeyWithValue("objectset.rio.cattle.io/id", "default-remove-resource"))
				g.Expect(cm.Annotations).To(HaveKey("meta.helm.sh/release-namespace"))
				g.Expect(cm.Labels).To(HaveKeyWithValue("app.kubernetes.io/managed-by", "Helm"))
				g.Expect(cm.Labels).To(HaveKey("objectset.rio.cattle.io/hash"))
			})
			// This is required for the resource to be watched.
			time.Sleep(5 * time.Second)
			deleteConfigMap("cm1")
			assertBundleDeployment("remove-resource", bundleDeploymentResourceMissing)
		})
	})

	When("all labels of a resource of a bundle deployment are removed", Label("remove-labels"), func() {
		It("should report that resource as \"not owned by us\"", func() {
			createBundleDeployment("remove-metadata", false)
			assertConfigMap(func(g Gomega, cm corev1.ConfigMap) {
				g.Expect(cm.Data).To(Equal(map[string]string{"key": "value"}))
				g.Expect(cm.Annotations).To(HaveKeyWithValue("meta.helm.sh/release-name", "remove-metadata"))
				g.Expect(cm.Annotations).To(HaveKeyWithValue("objectset.rio.cattle.io/id", "default-remove-metadata"))
				g.Expect(cm.Annotations).To(HaveKey("meta.helm.sh/release-namespace"))
				g.Expect(cm.Labels).To(HaveKeyWithValue("app.kubernetes.io/managed-by", "Helm"))
				g.Expect(cm.Labels).To(HaveKeyWithValue("objectset.rio.cattle.io/hash", "0f3e1d9d146fa8b290c0de403881184751430e59"))
			})
			updateConfigMap("cm1", func(cm *corev1.ConfigMap) {
				cm.Labels = map[string]string{}
			})
			assertBundleDeployment("remove-metadata", bundleDeploymentNotOwnedByUs)
		})
	})

	When("a bundle deployment adopts a \"clean\" resource", Label("clean"), func() {
		// A clean resource is a resource that does not bear labels or annotations indicating that it would belong to any other resource than our bundle deployment.
		It("verifies that the ConfigMap is adopted and its content merged", func() {
			createConfigMap(map[string]string{"foo": "bar"}, nil, nil)
			createBundleDeployment("adopt-clean", true)
			assertConfigMap(func(g Gomega, cm corev1.ConfigMap) {
				assertConfigMapAdopted(g, &cm)
				g.Expect(cm.Data).To(Equal(map[string]string{"foo": "bar", "key": "value"}))
			})
		})
	})

	When("a bundle deployment adopts a resource with wrangler metadata", Label("wrangler-metadata"), func() {
		It("verifies that the ConfigMap is adopted, its content merged and ownership changed", func() {
			const (
				objectSetHashKey   = "objectset.rio.cattle.io/hash"
				objectSetIDKey     = "objectset.rio.cattle.io/id"
				objectSetHashValue = "33ed67317c57ea78702e369c4c025f8df88553cc"
				objectSetIDValue   = "some-assumed-old-id"
			)
			createConfigMap(
				map[string]string{"foo": "bar"},
				map[string]string{objectSetHashKey: objectSetHashValue},
				map[string]string{objectSetIDKey: objectSetIDValue},
			)
			createBundleDeployment("adopt-wrangler-metadata", true)
			assertConfigMap(func(g Gomega, cm corev1.ConfigMap) {
				assertConfigMapAdopted(g, &cm)
				g.Expect(cm.Data).To(Equal(map[string]string{"foo": "bar", "key": "value"}))
				g.Expect(cm.Annotations).ToNot(HaveKeyWithValue(objectSetIDKey, objectSetIDValue))
				g.Expect(cm.Labels).ToNot(HaveKeyWithValue(objectSetHashKey, objectSetHashValue))
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
			assertConfigMap(func(g Gomega, cm corev1.ConfigMap) {
				assertConfigMapAdopted(g, &cm)
				g.Expect(cm.Data).To(Equal(map[string]string{"foo": "bar", "key": "value"}))
				g.Expect(cm.Annotations).ToNot(HaveKeyWithValue("objectset.rio.cattle.io/id", "$#@"))
				g.Expect(cm.Labels).ToNot(HaveKeyWithValue("objectset.rio.cattle.io/hash", "234"))
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
			assertConfigMap(func(g Gomega, cm corev1.ConfigMap) {
				assertConfigMapAdopted(g, &cm)
				g.Expect(cm.Data).To(Equal(map[string]string{"foo": "bar", "key": "value"}))
				g.Expect(cm.Annotations).To(HaveKeyWithValue("bar", "xzy"))
				g.Expect(cm.Labels).To(HaveKeyWithValue("foo", "234"))
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
