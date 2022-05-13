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

package controllers_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	securityv1alpha1 "nine.ch/sso/api/v1alpha1"
	"nine.ch/sso/controllers"
)

func TestOAuthProxyController(t *testing.T) {
	ctx := context.Background()
	client := setup(ctx, t)
	simple := newOAuthProxy(false)
	withIngress := newOAuthProxy(true)
	for _, p := range []securityv1alpha1.OAuthProxy{simple, withIngress} {
		if err := client.Create(ctx, &p); err != nil {
			t.Fatalf("could not create oauthProxy: %v", err)
		}

		if err := eventually(func() error {
			if err := client.Get(ctx, types.NamespacedName{Name: p.Name, Namespace: p.Namespace}, &p); err != nil {
				return err
			}
			if !p.Status.Ready {
				return fmt.Errorf("proxy is not ready")
			}
			return nil
		}, time.Minute, time.Second*2); err != nil {
			t.Fatalf("proxy did not get ready: %v", err)
		}

		if err := client.Delete(ctx, &p); err != nil {
			t.Fatalf("could not delete proxy: %v", err)
		}

		if err := eventually(func() error {
			nn := types.NamespacedName{p.Name, p.Namespace}
			if err := client.Get(ctx, nn, &appsv1.Deployment{}); err == nil || !apierrors.IsNotFound(err) {
				return fmt.Errorf("deployment still exists or other error %v", err)
			}

			if err := client.Get(ctx, nn, &networkingv1.Ingress{}); err == nil || !apierrors.IsNotFound(err) {
				return fmt.Errorf("ingress still exists or other error %v", err)
			}

			if err := client.Get(ctx, nn, &corev1.Service{}); err == nil || !apierrors.IsNotFound(err) {
				return fmt.Errorf("ingress still exists or other error %v", err)
			}
			return nil
		}, time.Minute, time.Second*2); err != nil {
			t.Fatalf("error waiting for all objects to be deleted: %v", err)
		}
	}
}

func eventually(f func() error, waitFor, tick time.Duration) error {
	timer := time.NewTimer(waitFor)
	defer timer.Stop()

	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	var err error
	for {
		select {
		case <-timer.C:
			return err
		case <-ticker.C:
			err = f()
			if err == nil {
				return nil
			}
		}
	}
}

func setup(ctx context.Context, t *testing.T) client.Client {
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&zap.Options{
		Development: true,
	})))
	testEnv := &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
		BinaryAssetsDirectory: "../test_assets/kubebuilder/bin",
	}

	cfg, err := testEnv.Start()
	if err != nil {
		t.Fatalf("could not start envtest: %v", err)
	}
	if err := securityv1alpha1.AddToScheme(scheme.Scheme); err != nil {
		t.Fatalf("could not add to scheme: %v", err)
	}

	client, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		t.Fatalf("could not setup client: %v", err)
	}

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	if err != nil {
		t.Fatalf("could not setup manager: %v", err)
	}

	if err := (&controllers.Oauthproxyreconciler{
		Client: k8sManager.GetClient(),
		Scheme: k8sManager.GetScheme(),
	}).SetupWithManager(k8sManager); err != nil {
		t.Fatalf("could not setup reconciler: %v", err)
	}

	go func() {
		if err := k8sManager.Start(ctx); err != nil {
			t.Errorf("could not start reconciler: %v", err)
		}
	}()

	t.Cleanup(func() {
		if err := testEnv.Stop(); err != nil {
			t.Errorf("could not stop envtest: %v", err)
		}
	})
	return client
}
