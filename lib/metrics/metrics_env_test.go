package metrics

import (
	"context"
	"encoding/json"

	"github.com/iter8-tools/etc3/api/v2alpha2"
	"github.com/iter8-tools/handler/base"
	"github.com/iter8-tools/handler/experiment"
	"github.com/iter8-tools/handler/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("metrics library", func() {
	Context("when running a metrics/collect task", func() {
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

		It("should initialize an experiment", func() {
			By("reading the experiment from file")
			exp, err = (&experiment.Builder{}).FromFile(utils.CompletePath("../../", "testdata/metricscollect/metricscollect.yaml")).Build()
			Expect(err).ToNot(HaveOccurred())

			By("creating experiment in cluster")
			Expect(k8sClient.Create(context.Background(), exp)).To(Succeed())

			By("getting the experiment from the cluster")
			exp2 := &experiment.Experiment{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{
				Namespace: "default",
				Name:      "metrics-collect-exp",
			}, exp2)).To(Succeed())

			By("populating context with the experiment")
			ctx := context.WithValue(context.Background(), base.ContextKey("experiment"), exp2)

			By("creating a metrics/collect task")
			ct := CollectTask{
				Library: "metrics",
				Task:    "collect",
				With: CollectInputs{
					Versions: []Version{
						{
							Name: "default",
							URL:  "https://httpbin.org",
						},
						{
							Name: "canary",
							URL:  "https://httpbin.org/stream/1",
						},
					},
				},
			}
			ct.InitializeDefaults()

			By("running the metrics/collect task")
			Expect(ct.Run(ctx)).ToNot(HaveOccurred())

			By("getting the experiment from cluster")
			exp3 := &experiment.Experiment{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{
				Namespace: "default",
				Name:      "metrics-collect-exp",
			}, exp3)).To(Succeed())

			By("confirming that the experiment looks right")
			Expect(exp3.Status.Analysis.AggregatedBuiltinHists).ToNot(BeNil())

			By("running the metrics/collect task again")
			Expect(ct.Run(ctx)).ToNot(HaveOccurred())

			By("getting the experiment from cluster")
			exp4 := &experiment.Experiment{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{
				Namespace: "default",
				Name:      "metrics-collect-exp",
			}, exp4)).To(Succeed())

			By("confirming that the experiment looks right")
			fortioData := make(map[string]*Result)

			Expect(exp4.Status.Analysis.AggregatedBuiltinHists).ToNot(BeNil())
			jsonBytes, err := json.Marshal(exp4.Status.Analysis.AggregatedBuiltinHists.Data)
			// convert jsonBytes to fortioData
			Expect(err).ShouldNot(HaveOccurred())
			err = json.Unmarshal(jsonBytes, &fortioData)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(fortioData).ToNot(BeNil())
			Expect(fortioData["default"]).ToNot(BeNil())
			Expect(fortioData["canary"]).ToNot(BeNil())
			Expect(fortioData["canary"].DurationHistogram.Count).To(Equal(80))
		})
	})
})
