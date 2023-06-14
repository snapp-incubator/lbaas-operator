/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"reflect"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	networkingv1alpha1 "github.com/snapp-incubator/lbaas-operator/api/v1alpha1"
)

const SkipReconcileAnnotation = "externalservice.networking.snappcloud.io/skip-reconcile"
const SkipReconcileDuration = 30 * time.Second

// ExternalServiceReconciler reconciles a ExternalService object
type ExternalServiceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=networking.snappcloud.io,resources=externalservices,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networking.snappcloud.io,resources=externalservices/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=networking.snappcloud.io,resources=externalservices/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups="",resources=endpoints,verbs=get;list;watch;create;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *ExternalServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger = logger.WithValues("Namespace", req.Namespace, "Name", req.Name)

	var externalService networkingv1alpha1.ExternalService
	if err := r.Get(ctx, req.NamespacedName, &externalService); err != nil {
		logger.Error(err, "could not get external service")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// This part of code if for use-cases that user wants to skip reconcile loop and make objects kind of unmanaged.
	_, shouldSkip := externalService.ObjectMeta.GetAnnotations()[SkipReconcileAnnotation]
	if shouldSkip {
		logger.Info("found " + SkipReconcileAnnotation + " in annotations. Skipping this reconcile loop.")
		return ctrl.Result{RequeueAfter: SkipReconcileDuration}, nil
	}

	desiredEndpoints := r.getDesiredEndpoints(externalService)

	var currentEndpoints corev1.Endpoints
	if err := r.Get(ctx, req.NamespacedName, &currentEndpoints); err != nil {
		err = r.Create(ctx, &desiredEndpoints)
		if err != nil {
			logger.Error(err, "could not create endpoints object")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	} else {
		// TODO: DeepEqual function does not operate good for this part because kubernetes adds values to fields after creation
		if !reflect.DeepEqual(&desiredEndpoints.Subsets, &currentEndpoints.Subsets) || !reflect.DeepEqual(&desiredEndpoints.Labels, &currentEndpoints.Labels) {
			err = r.Update(ctx, &desiredEndpoints)
			if err != nil {
				logger.Error(err, "could not update endpoints object")
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
	}

	desiredService := r.getDesiredService(externalService)

	var currentService corev1.Service
	if err := r.Get(ctx, req.NamespacedName, &currentService); err != nil {
		err = r.Create(ctx, &desiredService)
		if err != nil {
			logger.Error(err, "could not create service object")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	} else {
		// TODO: DeepEqual function does not operate good for this part because kubernetes adds values to fields after creation
		// we can check for more field differences here.
		isTypeEqual := reflect.DeepEqual(&desiredService.Spec.Type, &currentService.Spec.Type)
		areLabelsEqual := reflect.DeepEqual(&desiredService.Labels, &currentService.Labels)
		arePortsEqual := reflect.DeepEqual(&desiredService.Spec.Ports, &currentService.Spec.Ports)
		if !isTypeEqual || !areLabelsEqual || !arePortsEqual {
			err = r.Update(ctx, &desiredService)
			if err != nil {
				logger.Error(err, "could not update service object")
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
	}

	externalService.Status.ServiceAddress = fmt.Sprintf("%s.%s.svc.cluster.local", externalService.Name, req.Namespace)
	err := r.Status().Update(ctx, &externalService)
	if err != nil {
		logger.Error(err, "could not update status of external service")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *ExternalServiceReconciler) getDesiredEndpoints(externalService networkingv1alpha1.ExternalService) corev1.Endpoints {
	ports := make([]corev1.EndpointPort, 0, len(externalService.Spec.Ports))
	for _, port := range externalService.Spec.Ports {
		ports = append(ports, corev1.EndpointPort{
			Name:        port.Name,
			Port:        port.Port,
			Protocol:    port.Protocol,
			AppProtocol: port.AppProtocol,
		})
	}
	addresses := make([]corev1.EndpointAddress, 0, len(externalService.Spec.Static.Addresses))
	if externalService.Spec.Type == networkingv1alpha1.StaticExternalServiceType {
		for _, address := range externalService.Spec.Static.Addresses {
			addresses = append(addresses, corev1.EndpointAddress{
				IP: address.IP,
			})
		}
	}
	endpoints := corev1.Endpoints{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Endpoints",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      externalService.Name,
			Namespace: externalService.Namespace,
			Labels:    externalService.Labels,
		},
		Subsets: []corev1.EndpointSubset{
			{
				Addresses: addresses,
				Ports:     ports,
			},
		},
	}
	_ = ctrl.SetControllerReference(&externalService, &endpoints, r.Scheme)
	return endpoints
}

func (r *ExternalServiceReconciler) getDesiredService(externalService networkingv1alpha1.ExternalService) corev1.Service {
	ports := make([]corev1.ServicePort, 0, len(externalService.Spec.Ports))
	for _, port := range externalService.Spec.Ports {
		ports = append(ports, corev1.ServicePort{
			Name:        port.Name,
			Protocol:    port.Protocol,
			AppProtocol: port.AppProtocol,
			Port:        port.Port,
			TargetPort:  intstr.FromInt(int(port.Port)),
		})
	}
	svc := corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      externalService.Name,
			Namespace: externalService.Namespace,
			Labels:    externalService.Labels,
		},
		Spec: corev1.ServiceSpec{
			Ports:    ports,
			Selector: nil,
			Type:     externalService.Spec.ServiceType,
		},
	}
	_ = ctrl.SetControllerReference(&externalService, &svc, r.Scheme)
	return svc
}

// SetupWithManager sets up the controller with the Manager.
func (r *ExternalServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1alpha1.ExternalService{}).
		Owns(&corev1.Endpoints{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
