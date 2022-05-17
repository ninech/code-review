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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OAuthProxySpec defines the desired state of OAuthProxy
type OAuthProxySpec struct {
	// Service is a reference to the service that will be fronted
	// by this oauth proxy.
	Service corev1.LocalObjectReference `json:"service"`
	// Host is the hostname where the oauthProxy should be available at.
	// Only optional if an Ingress is specified.
	// +optional
	Host string `json:"host,omitempty"`
	// RedirectURL for the OIDC flow.
	RedirectURL string `json:"redirectURL"`
	// Ingress allows specifying the ingress. Overwrites host.
	// +optional
	Ingress *networkingv1.IngressSpec `json:"ingress,omitempty"`
}

// OAuthProxyStatus defines the observed state of OAuthProxy
type OAuthProxyStatus struct {
	// Ready indicates if the oauth proxy is ready or not.
	// +optional
	Ready bool `json:"ready,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// OAuthProxy is the Schema for the oauthproxies API
type OAuthProxy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OAuthProxySpec   `json:"spec,omitempty"`
	Status OAuthProxyStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OAuthProxyList contains a list of OAuthProxy
type OAuthProxyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OAuthProxy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OAuthProxy{}, &OAuthProxyList{})
}
