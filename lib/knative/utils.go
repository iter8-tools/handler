package knative

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/iter8-tools/etc3/api/v2alpha2"
	"github.com/iter8-tools/handler/experiment"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

// GetNamespacedNameForKsvc parses the target of the experiment and returns name and namespace of the ksvc.
func GetNamespacedNameForKsvc(e *experiment.Experiment) (*types.NamespacedName, error) {
	if nn := strings.Split(e.Spec.Target, "/"); len(nn) == 2 {
		return &types.NamespacedName{
			Namespace: nn[0],
			Name:      nn[1],
		}, nil
	}
	return nil, errors.New("unable to parse target into name and namespace of the ksvc")
}

// GetKnativeSvc fetches the Knative service given its name and namespace.
func GetKnativeSvc(nn *types.NamespacedName) (*servingv1.Service, error) {
	var err error
	log.Trace("Getting client")
	ksvc := &servingv1.Service{}
	log.Trace("Getting ksvc now...")
	if err = experiment.GetTypedObject(nn, ksvc); err == nil {
		log.Trace("got ksvc")
		return ksvc, nil
	}
	log.Error("cannot fetch ksvc from cluster")
	log.Error(err)
	return nil, err
}

// checkKsvcReadiness checks readiness of the knative service.
func checkKsvcReadiness(ksvc *servingv1.Service) error {
	var err = errors.New("knative service is not ready")
	ksvc1 := ksvc
	log.Trace("ksvc status...")
	log.Trace(ksvc.Status)
	for i := 0; i < experiment.NumAttempt; i++ {
		a, b, c := false, false, false
		log.Trace(ksvc1.Status.Conditions)
		for _, condition := range ksvc1.Status.Conditions {
			if condition.Type == servingv1.ServiceConditionReady &&
				condition.Status == corev1.ConditionTrue {
				a = true
			}
			if condition.Type == servingv1.ServiceConditionConfigurationsReady &&
				condition.Status == corev1.ConditionTrue {
				b = true
			}
			if condition.Type == servingv1.ServiceConditionRoutesReady &&
				condition.Status == corev1.ConditionTrue {
				c = true
			}
		}
		// everything looks good
		if a && b && c {
			return nil
		}
		// not ready yet
		time.Sleep(experiment.Period)
		ksvc1 = &servingv1.Service{}
		if err = experiment.GetTypedObject(&types.NamespacedName{
			Namespace: ksvc.Namespace,
			Name:      ksvc.Name,
		}, ksvc1); err != nil {
			log.Error(err)
			return errors.New("unable to get ksvc")
		}
	}
	return err
}

// updateLocalExp updates the given Knative experiment struct.
func updateLocalExp(e *experiment.Experiment, ksvc *servingv1.Service) error {
	switch e.Spec.Strategy.TestingPattern {
	case v2alpha2.TestingPatternConformance:
		return updateLocalConformanceExp(e, ksvc)
	case v2alpha2.TestingPatternCanary:
		return updateLocalCanaryExp(e, ksvc)
	default:
		return errors.New("unsupported testing pattern found in experiment")
	}
}

// updateLocalConformanceExp updates the given Knative conformance experiment struct.
func updateLocalConformanceExp(e *experiment.Experiment, ksvc *servingv1.Service) error {
	if e.Spec.VersionInfo == nil {
		return errors.New("nil valued VersionInfo in experiment")
	}
	if revision, err := findRevisionInVersionDetail(&e.Spec.VersionInfo.Baseline); err == nil {
		if revisionPresentInKsvc(revision, ksvc) {
			experiment.UpdateVariable(&e.Spec.VersionInfo.Baseline, "namespace", ksvc.Namespace)
			return nil
		}
		return errors.New("unable to find revision in ksvc")
	}
	return errors.New("unable to find revision information in conformance experiment")
}

// findRevisionFromVersionDetail finds the name of the revision in the given VersionDetail struct.
func findRevisionInVersionDetail(v *v2alpha2.VersionDetail) (string, error) {
	return experiment.FindVariableInVersionDetail(v, "revision")
}

// revisionPresentInKsvc checks if the given revision is present in the ksvc.
func revisionPresentInKsvc(revision string, ksvc *servingv1.Service) bool {
	log.Trace("revisionPresentInKsvc invoked")
	if ksvc == nil {
		log.Error("nil ksvc")
		return false
	}
	for i := 0; i < len(ksvc.Status.Traffic); i++ {
		if ksvc.Status.Traffic[i].RevisionName == revision {
			log.Trace("returning true... revision found")
			return true
		}
	}
	log.Error("no match found for revision name: ", revision)
	log.Error("traffic targets", ksvc.Status.Traffic)
	return false
}

// updateLocalCanaryExp updates the given Knative canary experiment struct.
func updateLocalCanaryExp(e *experiment.Experiment, ksvc *servingv1.Service) error {
	var err error
	if e == nil || ksvc == nil {
		return errors.New("experiment and ksvc cannot be nil valued")
	}
	if e.Spec.VersionInfo == nil {
		return errors.New("baseline absent in canary experiment")
	}
	if len(e.Spec.VersionInfo.Candidates) != 1 {
		return errors.New("canary experiment needs to have a single candidate")
	}

	// find baseline revision; update baseline with namespace
	var br string
	if br, err = findRevisionInVersionDetail(&e.Spec.VersionInfo.Baseline); err == nil {
		experiment.UpdateVariable(&e.Spec.VersionInfo.Baseline, "namespace", ksvc.Namespace)
		// find status.trafficTarget for baseline
		var tt *servingv1.TrafficTarget
		if tt, err = findTrafficTargetInStatus(ksvc, br); err == nil {
			var tti int
			// set its weightObjRef
			if tti, err = findTrafficTargetIndexInSpec(tt, ksvc); err == nil {
				e.Spec.VersionInfo.Baseline.WeightObjRef = objRefFromFieldPath(tti, ksvc)
			}
		}
	}

	// repeat the above steps for baseline for candidate as well
	// find candidate revision; update candidate with namespace
	var cr string
	if cr, err = findRevisionInVersionDetail(&e.Spec.VersionInfo.Candidates[0]); err == nil {
		experiment.UpdateVariable(&e.Spec.VersionInfo.Candidates[0], "namespace", ksvc.Namespace)
		// find status.trafficTarget for candidate
		var tt *servingv1.TrafficTarget
		if tt, err = findTrafficTargetInStatus(ksvc, cr); err == nil {
			var tti int
			// set its weightObjRef
			if tti, err = findTrafficTargetIndexInSpec(tt, ksvc); err == nil {
				e.Spec.VersionInfo.Candidates[0].WeightObjRef = objRefFromFieldPath(tti, ksvc)
			}
		}
	}

	return err
}

// findTrafficTargetInStatus finds the traffic target corresponding to the given revision in ksvc's status.
func findTrafficTargetInStatus(ksvc *servingv1.Service, revision string) (*servingv1.TrafficTarget, error) {
	if ksvc == nil {
		return nil, errors.New("nil valued ksvc")
	}
	for i := 0; i < len(ksvc.Status.Traffic); i++ {
		if ksvc.Status.Traffic[i].RevisionName == revision {
			return &ksvc.Status.Traffic[i], nil
		}
	}
	return nil, errors.New("unable to find traffic target for given revision in ksvc status")
}

// objRefFromFieldPath returns a weightObjRef to be inserted into the experiment using the given fieldPath fragment
func objRefFromFieldPath(fieldPathIndex int, ksvc *servingv1.Service) *corev1.ObjectReference {
	apiVersion, kind := ksvc.GetGroupVersionKind().ToAPIVersionAndKind()
	return &corev1.ObjectReference{
		Kind:       kind,
		Namespace:  ksvc.Namespace,
		Name:       ksvc.Name,
		APIVersion: apiVersion,
		FieldPath:  ".spec.traffic[" + fmt.Sprint(fieldPathIndex) + "].percent",
	}
}

// findTrafficTargetIndexInSpec finds the index of the given traffic target within the spec.traffic stanza of the given ksvc.
func findTrafficTargetIndexInSpec(tt *servingv1.TrafficTarget, ksvc *servingv1.Service) (int, error) {
	for i := 0; i < len(ksvc.Spec.Traffic); i++ {
		if ksvc.Spec.Traffic[i].RevisionName == tt.RevisionName {
			// matched by revisionName property
			return i, nil
		}
		if ksvc.Spec.Traffic[i].LatestRevision != nil {
			if *ksvc.Spec.Traffic[i].LatestRevision == true {
				if tt.LatestRevision != nil {
					if *tt.LatestRevision == true {
						// matched by latestRevision property
						return i, nil
					}
				}
			}
		}
	}
	return -1, errors.New("unable to find traffic target in spec")
}
