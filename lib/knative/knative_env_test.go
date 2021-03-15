package knative

import (
	"context"
	"io/ioutil"

	"github.com/ghodss/yaml"
	"github.com/iter8-tools/etc3/api/v2alpha2"
	"github.com/iter8-tools/handler/base"
	"github.com/iter8-tools/handler/experiment"
	"github.com/iter8-tools/handler/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Knative library", func() {
	Context("when running a knative experiment", func() {
		var exp *experiment.Experiment
		var err error

		u1 := &unstructured.Unstructured{}
		u1.SetGroupVersionKind((&servingv1.Service{}).GetGroupVersionKind())
		u2 := &unstructured.Unstructured{}
		u2.SetGroupVersionKind(v2alpha2.GroupVersion.WithKind("experiment"))
		BeforeEach(func() {
			k8sClient.DeleteAllOf(context.Background(), u1, client.InNamespace("default"))
			k8sClient.DeleteAllOf(context.Background(), u2, client.InNamespace("default"))
		})

		It("should initialize a conformance experiment", func() {
			By("reading the experiment from file")
			exp, err = (&experiment.Builder{}).FromFile(utils.CompletePath("../../", "testdata/knative/conformanceexp.yaml")).Build()
			Expect(err).ToNot(HaveOccurred())

			By("creating experiment in cluster")
			Expect(k8sClient.Create(context.Background(), exp)).To(Succeed())

			By("reading the Knative service from file")
			ksvc := &servingv1.Service{}
			data, err := ioutil.ReadFile(utils.CompletePath("../../", "testdata/knative/onerevision.yaml"))
			Expect(err).ToNot(HaveOccurred())
			err = yaml.Unmarshal(data, ksvc)
			Expect(err).ToNot(HaveOccurred())

			By("creating the service in the cluster")
			// deep copying.. in order to retain status in original ksvc
			ksvcCopy := ksvc.DeepCopy()
			ksvcCopy.ResourceVersion = ""
			Expect(k8sClient.Create(context.Background(), ksvcCopy)).To(Succeed())

			By("updating the status of the service in the cluster")
			ksvc.Status.DeepCopyInto(&ksvcCopy.Status)
			Expect(k8sClient.Status().Update(context.Background(), ksvcCopy)).To(Succeed())

			By("getting the experiment from the cluster")
			exp2 := &experiment.Experiment{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{
				Namespace: "default",
				Name:      "experiment-1",
			}, exp2)).To(Succeed())

			By("populating context with the experiment")
			ctx := context.WithValue(context.Background(), base.ContextKey("experiment"), exp2)

			By("creating an init-experiment task")
			initExp := InitExperimentTask{
				Library: "knative",
				Task:    "init-experiment",
			}

			By("running the init-experiment task")
			Expect(initExp.Run(ctx)).ToNot(HaveOccurred())

			By("getting the experiment from cluster")
			exp3 := &experiment.Experiment{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{
				Namespace: "default",
				Name:      "experiment-1",
			}, exp3)).To(Succeed())

			By("confirming that the experiment looks right")
			Expect(exp3.Spec.VersionInfo).ToNot(BeNil())
		})

		It("should initialize a canary experiment", func() {
			By("reading the experiment from file")
			exp, err = (&experiment.Builder{}).FromFile(utils.CompletePath("../../", "testdata/knative/canaryexp.yaml")).Build()
			Expect(err).ToNot(HaveOccurred())

			By("creating experiment in cluster")
			Expect(k8sClient.Create(context.Background(), exp)).To(Succeed())

			By("reading the Knative service from file")
			ksvc := &servingv1.Service{}
			data, err := ioutil.ReadFile(utils.CompletePath("../../", "testdata/knative/tworevisions.yaml"))
			Expect(err).ToNot(HaveOccurred())
			err = yaml.Unmarshal(data, ksvc)
			Expect(err).ToNot(HaveOccurred())

			By("creating the service in the cluster")
			// deep copying.. in order to retain status in original ksvc
			ksvcCopy := ksvc.DeepCopy()
			ksvcCopy.ResourceVersion = ""
			Expect(k8sClient.Create(context.Background(), ksvcCopy)).To(Succeed())

			By("updating the status of the service in the cluster")
			ksvc.Status.DeepCopyInto(&ksvcCopy.Status)
			Expect(k8sClient.Status().Update(context.Background(), ksvcCopy)).To(Succeed())

			By("getting the experiment from the cluster")
			exp2 := &experiment.Experiment{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{
				Namespace: "default",
				Name:      "experiment-1",
			}, exp2)).To(Succeed())

			By("populating context with the experiment")
			ctx := context.WithValue(context.Background(), base.ContextKey("experiment"), exp2)

			By("creating an init-experiment task")
			initExp := InitExperimentTask{
				Library: "knative",
				Task:    "init-experiment",
			}

			By("running the init-experiment task")
			Expect(initExp.Run(ctx)).ToNot(HaveOccurred())

			By("getting the experiment from cluster")
			exp3 := &experiment.Experiment{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{
				Namespace: "default",
				Name:      "experiment-1",
			}, exp3)).To(Succeed())

			By("confirming that the experiment looks right")
			Expect(exp3.Spec.VersionInfo).ToNot(BeNil())
			Expect(exp3.Spec.VersionInfo.Baseline.WeightObjRef).ToNot(BeNil())
			Expect(*exp3.Spec.VersionInfo.Baseline.WeightObjRef).To(Equal(corev1.ObjectReference{
				Kind:       ksvc.Kind,
				Namespace:  ksvc.Namespace,
				Name:       ksvc.Name,
				APIVersion: ksvc.APIVersion,
				FieldPath:  ".spec.traffic[0].percent",
			}))
			Expect(exp3.Spec.VersionInfo.Candidates).ToNot(BeEmpty())
			Expect(exp3.Spec.VersionInfo.Candidates[0].WeightObjRef).ToNot(BeNil())
			Expect(*exp3.Spec.VersionInfo.Candidates[0].WeightObjRef).To(Equal(corev1.ObjectReference{
				Kind:       ksvc.Kind,
				Namespace:  ksvc.Namespace,
				Name:       ksvc.Name,
				APIVersion: ksvc.APIVersion,
				FieldPath:  ".spec.traffic[1].percent",
			}))
		})
	})
})
