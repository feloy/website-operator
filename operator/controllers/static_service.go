package controllers

import (
	"context"
	"fmt"

	websitev1alpha1 "example.com/website/v1alpha1/api/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *StaticReconciler) applyService(ctx context.Context, log logr.Logger, static *websitev1alpha1.Static) error {

	// Create in memory the Service that is expected to exist into cluster
	expected := r.createService(static)

	// Get the existing Service from cluster, if any
	found := new(corev1.Service)
	err := r.Get(ctx, types.NamespacedName{Name: expected.ObjectMeta.Name, Namespace: expected.ObjectMeta.Namespace}, found)
	if err == nil {
		// Service exists in cluster

		// Check status
		if len(found.Status.LoadBalancer.Ingress) > 0 && found.Status.LoadBalancer.Ingress[0].IP != static.Status.ExternalIP {
			log.Info(fmt.Sprintf("New external IP: %s", found.Status.LoadBalancer.Ingress[0].IP))
			static.Status.ExternalIP = found.Status.LoadBalancer.Ingress[0].IP
			err = r.Status().Update(ctx, static)
			if err != nil {
				return err
			}
		}

		expected.Spec.Ports[0].NodePort = found.Spec.Ports[0].NodePort // Do not check nodeport
		if !equality.Semantic.DeepDerivative(expected.Spec, found.Spec) {

			log.Info("Service found but different than expected => Update service")

			controllerutil.SetControllerReference(static, expected, r.Scheme)
			r.updateService(static, found)
			err = r.Update(ctx, found)
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

	log.Info("Service not found => Create service")

	// Set static as parent of service
	controllerutil.SetControllerReference(static, expected, r.Scheme)

	if err = r.Create(ctx, expected); err != nil {
		log.Error(err, "unable to create service for static")
	}

	return nil
}

func (r *StaticReconciler) createService(static *websitev1alpha1.Static) *corev1.Service {
	name := static.Name + "-service"

	result := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{

			Name:      name,
			Namespace: static.Namespace,
		},

		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{},
			},
		},
	}
	r.updateService(static, result)
	return result
}

func (r *StaticReconciler) updateService(static *websitev1alpha1.Static, service *corev1.Service) {
	service.Spec.Type = corev1.ServiceTypeLoadBalancer
	service.Spec.Selector = map[string]string{
		"app": static.Name + "-deployment",
	}
	if len(service.Spec.Ports) > 0 {
		service.Spec.Ports[0].Port = 80
		service.Spec.Ports[0].TargetPort = intstr.FromInt(80)
		service.Spec.Ports[0].Protocol = corev1.ProtocolTCP
	}
}
