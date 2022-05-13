/*
Copyright 2022.

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
	"time"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	securityv1alpha1 "nine.ch/sso/api/v1alpha1"
)

// Oauthproxyreconciler reconciles a OAuthProxy object
type Oauthproxyreconciler struct {
	client.Client
	Scheme *runtime.Scheme

	OIDC *KeycloakClient
}

//+kubebuilder:rbac:groups=security.nine.ch,resources=oauthproxies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=security.nine.ch,resources=oauthproxies/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=security.nine.ch,resources=oauthproxies/finalizers,verbs=update

func (x *Oauthproxyreconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log = log.WithValues("request", req.NamespacedName)
	log.Info("handling request")

	var oauthProxy securityv1alpha1.OAuthProxy
	if err := x.Get(ctx, req.NamespacedName, &oauthProxy); err != nil {
		log.Error(err, "unable to fetch OAuthProxy")
		// Ignore not founds.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	oidcCfg, err := x.OIDC.GetOIDCConfig(req.Name+"_"+req.Namespace, oauthProxy.Spec.RedirectURL)
	if err != nil {
		log.Error(err, "could not get oidc config")
		return ctrl.Result{}, errors.Wrapf(err, "could not get oidc config")
	}

	expectedDeployment := OAuthProxyDeployment(oauthProxy, oidcCfg)
	if err := ctrl.SetControllerReference(&oauthProxy, expectedDeployment, x.Scheme); err != nil {
		log.Error(err, "unable to set deployment's owner reference")
		return ctrl.Result{}, err
	}

	var proxy appsv1.Deployment
	if err := x.Get(ctx, req.NamespacedName, &proxy); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, fmt.Errorf("could not get deployment")
		}
		log.Info("creating deployment")
		return requeueWithDelay(), x.Create(ctx, expectedDeployment)
	}

	if !compareDeployment(&proxy, expectedDeployment) {
		log.Info("not equal", "a", proxy, "b", *expectedDeployment)
		if err := x.Update(ctx, expectedDeployment); err != nil {
			log.Error(err, "unable to update oauthProxy deployment")
			return ctrl.Result{}, err
		}

		return requeueWithDelay(), nil
	}

	expectedIngress := OAuthProxyIngress(oauthProxy)
	if err := ctrl.SetControllerReference(&oauthProxy, expectedIngress, x.Scheme); err != nil {
		log.Error(err, "unable to set deployment's owner reference")
		return ctrl.Result{}, err
	}
	var ingress networkingv1.Ingress
	if err := x.Get(ctx, req.NamespacedName, &ingress); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, fmt.Errorf("could not get ingress")
		}
		log.Info("creating ingress")
		return requeueWithDelay(), x.Create(ctx, expectedIngress)
	}

	if !compareIngress(&ingress, expectedIngress) {
		log.Info("not equal", "a", ingress, "b", *expectedIngress)
		if err := x.Update(ctx, expectedIngress); err != nil {
			log.Error(err, "unable to update oauthProxy ingress")
			return ctrl.Result{}, err
		}

		return requeueWithDelay(), nil
	}

	expectedService := OAuthProxyService(oauthProxy)
	if err := ctrl.SetControllerReference(&oauthProxy, expectedService, x.Scheme); err != nil {
		log.Error(err, "unable to set deployment's owner reference")
		return ctrl.Result{}, err
	}
	var service corev1.Service
	if err := x.Get(ctx, req.NamespacedName, &service); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, fmt.Errorf("could not get service")
		}
		log.Info("creating service")
		return requeueWithDelay(), x.Create(ctx, expectedService)
	}

	if !compareService(&service, expectedService) {
		log.Info("not equal", "a", service, "b", *expectedService)
		if err := x.Update(ctx, expectedService); err != nil {
			log.Error(err, "unable to update oauthProxy service")
			return ctrl.Result{}, err
		}

		return requeueWithDelay(), nil
	}

	log.Info("proxy is ready")

	oauthProxy.Status.Ready = true
	if err := x.Status().Update(ctx, &oauthProxy); err != nil {
		log.Error(err, "unable to update oauthProxy status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func requeueWithDelay() ctrl.Result {
	return ctrl.Result{
		RequeueAfter: 200 * time.Millisecond,
	}
}

type OidcConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

func compareDeployment(a, b *appsv1.Deployment) bool {
	if a.Name != b.Name || a.Namespace != b.Namespace {
		return false
	}

	if len(a.Spec.Selector.MatchLabels) != len(b.Spec.Selector.MatchLabels) {
		return false
	}
	for k, v := range a.Spec.Selector.MatchLabels {
		if vb, ok := b.Spec.Selector.MatchLabels[k]; !ok || vb != v {
			return false
		}
	}

	if *a.Spec.Replicas != *b.Spec.Replicas {
		return false
	}

	if len(a.Spec.Template.Spec.Containers) != len(b.Spec.Template.Spec.Containers) {
		return false
	}
	ac := a.Spec.Template.Spec.Containers[0]
	bc := b.Spec.Template.Spec.Containers[0]
	for i, arg := range ac.Args {
		if bc.Args[i] != arg {
			return false
		}
	}
	return true
}

// OAuthProxyDeployment returns a deployment containing oauthProxy with the
// configuration from the given CR.
func OAuthProxyDeployment(cr securityv1alpha1.OAuthProxy, cfg *OidcConfig) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":      "oauth-proxy",
					"instance": cr.Name,
				},
			},
			Replicas: pointer.Int32(1),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":      "oauth-proxy",
						"instance": cr.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "oauth-proxy",
						Image: "gcr.io/our-oauth-proxy-image:latest",
						Args: []string{
							"-target-service=" + cr.Spec.Service.Name,
							"-client-id=" + cfg.ClientID,
							"-client-secret=" + cfg.ClientSecret,
							"-redirect-url=" + cfg.RedirectURL,
						},
						Ports: []corev1.ContainerPort{{
							Name:          "http",
							ContainerPort: 80,
						}},
					}},
				},
			},
		},
	}
}

func compareIngress(a, b *networkingv1.Ingress) bool {
	if a.Name != b.Name || a.Namespace != b.Namespace {
		return false
	}

	if len(a.Annotations) != len(b.Annotations) {
		return false
	}
	for k, v := range a.Annotations {
		if b.Annotations[k] != v {
			return false
		}
	}

	if len(a.Spec.TLS) != len(b.Spec.TLS) {
		return false
	}

	if len(a.Spec.TLS[0].Hosts) != len(b.Spec.TLS[0].Hosts) {
		return false
	}
	if a.Spec.TLS[0].Hosts[0] != b.Spec.TLS[0].Hosts[0] {
		return false
	}

	if len(a.Spec.Rules) != len(b.Spec.Rules) {
		return false
	}
	ar := a.Spec.Rules[0]
	br := b.Spec.Rules[0]
	if ar.Host != br.Host {
		return false
	}

	if len(ar.IngressRuleValue.HTTP.Paths) != len(br.IngressRuleValue.HTTP.Paths) {
		return false
	}

	ap := ar.IngressRuleValue.HTTP.Paths[0]
	bp := ar.IngressRuleValue.HTTP.Paths[0]
	if ap.Path != bp.Path {
		return false
	}
	if ap.Backend.Service.Name != bp.Backend.Service.Name || ap.Backend.Service.Port.Name != bp.Backend.Service.Port.Name {
		return false
	}

	return true
}

func OAuthProxyIngress(cr securityv1alpha1.OAuthProxy) *networkingv1.Ingress {
	pathType := networkingv1.PathTypeImplementationSpecific
	return &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Annotations: map[string]string{
				// we want TLS certificates for this ingress.
				"kubernetes.io/tls-acme": "true",
			},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Ingress",
			APIVersion: "networking.k8s.io/v1",
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: nil, // we expect a default ingress class to exist.
			TLS: []networkingv1.IngressTLS{{
				Hosts:      []string{cr.Spec.Host},
				SecretName: cr.Name + "-tls",
			}},
			Rules: []networkingv1.IngressRule{{
				Host: cr.Spec.Host,
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{{
							Path:     "/",
							PathType: &pathType,
							Backend: networkingv1.IngressBackend{
								Service: &networkingv1.IngressServiceBackend{
									Name: cr.Name,
									Port: networkingv1.ServiceBackendPort{
										Name: "http",
									},
								},
							},
						}},
					},
				},
			}},
		},
	}
}

func compareService(a, b *corev1.Service) bool {
	if a.Name != b.Name || a.Namespace != b.Namespace {
		return false
	}

	if len(a.Spec.Selector) != len(b.Spec.Selector) {
		return false
	}
	for k, v := range a.Spec.Selector {
		if vb, ok := b.Spec.Selector[k]; !ok || vb != v {
			return false
		}
	}

	if len(a.Spec.Ports) != len(b.Spec.Ports) {
		return false
	}
	ap := a.Spec.Ports[0]
	bp := b.Spec.Ports[0]
	if ap.Name != bp.Name || ap.Port != bp.Port || ap.TargetPort != bp.TargetPort {
		return false
	}
	return true
}
func OAuthProxyService(cr securityv1alpha1.OAuthProxy) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind: "Service",
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app":      "oauth-proxy",
				"instance": cr.Name,
			},
			Ports: []corev1.ServicePort{{
				Name:       "http",
				Port:       80,
				TargetPort: intstr.FromString("http"),
			}},
		},
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *Oauthproxyreconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&securityv1alpha1.OAuthProxy{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}
