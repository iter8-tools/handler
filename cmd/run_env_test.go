package cmd

import (
	"context"
	"os"

	"github.com/iter8-tools/handler/experiment"
	"github.com/iter8-tools/handler/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Experiment's handler field", func() {
	var exp *experiment.Experiment
	var err error
	var head = func() {
		By("reading the experiment from file")
		exp, err = (&experiment.Builder{}).FromFile(utils.CompletePath("../", "testdata/experiment6.yaml")).Build()
		Expect(err).ToNot(HaveOccurred())
	}
	var create = func() {
		By("creating experiment in cluster")
		Expect(k8sClient.Create(context.Background(), exp)).To(Succeed())
		Expect(k8sClient.Status().Update(context.Background(), exp)).To(Succeed())
	}
	var runhandler = func(by string) {
		By(by)
		os.Setenv("EXPERIMENT_NAME", "sklearn-iris-experiment-6")
		os.Setenv("EXPERIMENT_NAMESPACE", "default")
		name, namespace, err := getExperimentNN()
		Expect(err).ToNot(HaveOccurred())
		Expect("sklearn-iris-experiment-6").To(Equal(name))
		Expect("default").To(Equal(namespace))

		action = "start"
		runCmd.Run(nil, nil)
	}
	var tail = func(runby string) {
		create()
		runhandler(runby)
		Expect(k8sClient.Delete(context.Background(), exp)).To(Succeed())
	}

	Context("when containing handler actions", func() {
		It("should run handler", func() {
			head()
			tail("as a normal run")
		})
	})
	Context("when not containing the specified action", func() {
		It("should exit gracefully", func() {
			head()
			delete(*exp.Spec.Strategy.Handlers.Actions, "start")
			tail("with a warning")
		})
		It("should exit gracefully when ActionMap is nil", func() {
			head()
			exp.Spec.Strategy.Handlers.Actions = nil
			tail("with a warning")
		})
		It("should exit gracefully when handlers is nil", func() {
			head()
			exp.Spec.Strategy.Handlers = nil
			tail("with a warning")
		})
	})
})
