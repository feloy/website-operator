package controllers

import (
	"context"
	"fmt"

	websitev1alpha1 "example.com/website/v1alpha1/api/v1alpha1"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *StaticReconciler) applyDeployment(ctx context.Context, log logr.Logger, static *websitev1alpha1.Static) error {

	// Create in memory the Deployment that is expected to exist into cluster
	expected := r.createDeployment(static)

	// Get the existing Deployment from cluster, if any
	found := new(appsv1.Deployment)
	err := r.Get(ctx, types.NamespacedName{Name: expected.ObjectMeta.Name, Namespace: expected.ObjectMeta.Namespace}, found)
	if err == nil {
		// Deployment exists in cluster

		// Check status
		if found.Status.Replicas != static.Status.Replicas {
			log.Info(fmt.Sprintf("New Replicas: %d", found.Status.Replicas))
			static.Status.Replicas = found.Status.Replicas
			err = r.Status().Update(ctx, static)
			if err != nil {
				return err
			}
			r.Recorder.Eventf(static, corev1.EventTypeNormal, "update-replicas", "The replicas has been updated to %d", static.Status.Replicas)
		}

		if !equality.Semantic.DeepDerivative(expected.Spec, found.Spec) {

			log.Info("Deployment found but different than expected => Update deployment")

			controllerutil.SetControllerReference(static, expected, r.Scheme)
			err = r.Update(ctx, expected)
			if err != nil {
				return err
			}

			r.Recorder.Eventf(static, corev1.EventTypeNormal, "update-deployment", "The deployment '%s.%s' has been updated due to unexpected change", expected.Namespace, expected.Name)
		}
		return nil
	}

	err = client.IgnoreNotFound(err)
	if err != nil {
		// Error trying to get
		return err
	}

	log.Info("Deployment not found => Create deployment")

	// Set static as parent of deployment
	controllerutil.SetControllerReference(static, expected, r.Scheme)

	if err = r.Create(ctx, expected); err != nil {
		log.Error(err, "unable to create deployment for static")
	}

	r.Recorder.Eventf(static, corev1.EventTypeNormal, "create-deployment", "The deployment '%s.%s' has been created", expected.Namespace, expected.Name)

	return nil
}

func (r *StaticReconciler) createDeployment(static *websitev1alpha1.Static) *appsv1.Deployment {
	name := static.Name + "-deployment"
	labels := map[string]string{
		"app": name,
	}
	volumeName := "static-files"

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{

			Name:      name,
			Namespace: static.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: volumeName,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									SizeLimit: &static.Spec.DiskSize,
								},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:  "copy-static-files",
							Image: "gcr.io/cloud-builders/gcloud",
							Command: []string{
								"bash",
								"-c",
								"gsutil cp -R $(SOURCE)/* /mnt/",
							},
							Env: []corev1.EnvVar{
								{
									Name:  "SOURCE",
									Value: static.Spec.Source,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									MountPath: "/mnt",
									Name:      volumeName,
									ReadOnly:  false,
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:  "website",
							Image: "nginx",
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									"memory": *resource.NewQuantity(r.Config.MemoryRequestMi*1024*1024, resource.BinarySI),
									"cpu":    *resource.NewMilliQuantity(r.Config.CpuRequestMilli, resource.DecimalSI),
								},
								Limits: corev1.ResourceList{
									"memory": *resource.NewQuantity(r.Config.MemoryLimitMi*1024*1024, resource.BinarySI),
									"cpu":    *resource.NewMilliQuantity(r.Config.CpuLimitMilli, resource.DecimalSI),
								},
							},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 80,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									MountPath: "/usr/share/nginx/html",
									Name:      volumeName,
									ReadOnly:  true,
								},
							},
						},
					},
				},
			},
		},
	}
}
