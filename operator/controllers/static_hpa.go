package controllers

import (
	"context"

	websitev1alpha1 "example.com/website/v1alpha1/api/v1alpha1"
	"github.com/go-logr/logr"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *StaticReconciler) applyHPA(ctx context.Context, log logr.Logger, static *websitev1alpha1.Static) error {

	// Create in memory the HPA that is expected to exist into cluster
	expected := r.createHPA(static)

	// Get the existing HPA from cluster, if any
	found := new(autoscalingv1.HorizontalPodAutoscaler)
	err := r.Get(ctx, types.NamespacedName{Name: expected.ObjectMeta.Name, Namespace: expected.ObjectMeta.Namespace}, found)
	if err == nil {
		// HPA exists in cluster

		if !equality.Semantic.DeepDerivative(expected.Spec, found.Spec) {

			log.Info("HPA found but different than expected => Update HPA")

			controllerutil.SetControllerReference(static, expected, r.Scheme)
			err = r.Update(ctx, expected)
			if err != nil {
				return err
			}
		}
		return nil
	}

	err = client.IgnoreNotFound(err)
	if err != nil {
		// Error trying to get
		return err
	}

	log.Info("HPA not found => Create HPA")

	// Set static as parent of HPA
	controllerutil.SetControllerReference(static, expected, r.Scheme)

	if err = r.Create(ctx, expected); err != nil {
		log.Error(err, "unable to create HPA for static")
	}

	return nil
}

func (r *StaticReconciler) createHPA(static *websitev1alpha1.Static) *autoscalingv1.HorizontalPodAutoscaler {
	name := static.Name + "-hpa"

	return &autoscalingv1.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{

			Name:      name,
			Namespace: static.Namespace,
		},
		Spec: autoscalingv1.HorizontalPodAutoscalerSpec{
			MinReplicas:                    &static.Spec.MinReplicas,
			MaxReplicas:                    static.Spec.MaxReplicas,
			TargetCPUUtilizationPercentage: &r.Config.CpuUtilization,
			ScaleTargetRef: autoscalingv1.CrossVersionObjectReference{
				Kind:       "Deployment",
				APIVersion: "apps/v1",
				Name:       static.Name + "-deployment",
			},
		},
	}
}
