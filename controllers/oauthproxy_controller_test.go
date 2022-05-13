package controllers_test

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	securityv1alpha1 "nine.ch/sso/api/v1alpha1"
	"nine.ch/sso/controllers"
)

func TestOAuthProxyDeployment(t *testing.T) {
	p := newOAuthProxy()
	cfg := &controllers.OidcConfig{
		ClientSecret: "secret",
		ClientID:     "a-b-c-d",
		RedirectURL:  "https://example.com",
	}
	deploy := controllers.OAuthProxyDeployment(p, cfg)
	if *deploy.Spec.Replicas != 1 {
		t.Fatalf("incorrect amount of replicas")
	}

	if len(deploy.Spec.Template.Spec.Containers) != 1 {
		t.Fatalf("incorrect amount of containers")
	}

	if deploy.Spec.Template.Spec.Containers[0].Args[3] != "-redirect-url=https://example.com" {
		t.Fatalf("incorrect redirect URL")
	}
}

func TestOAuthProxyIngress(t *testing.T) {
	p := newOAuthProxy()
	ingress := controllers.OAuthProxyIngress(p)
	if ingress.Annotations["kubernetes.io/tls-acme"] != "true" {
		t.Fatalf("incorrect TLS annotation")
	}

	if len(ingress.Spec.TLS) != 1 {
		t.Fatalf("no TLS config specified")
	}

	if len(ingress.Spec.Rules) != 1 {
		t.Fatalf("no rules specified")
	}

	if ingress.Spec.Rules[0].Host != p.Spec.Host {
		t.Fatalf("wrong host on ingress")
	}
}

func newOAuthProxy() securityv1alpha1.OAuthProxy {
	return securityv1alpha1.OAuthProxy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: securityv1alpha1.OAuthProxySpec{
			Service: corev1.LocalObjectReference{
				Name: "my-service",
			},
			Host:        "my-cool-domain.com",
			RedirectURL: "https://my-cool-domain.com/authorization",
		},
	}
}
