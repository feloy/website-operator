/*


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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// StaticSpec defines the desired state of Static
type StaticSpec struct {
	// DiskSize indicates the amount of disk space to reserve to store assets for each instance
	DiskSize resource.Quantity `json:"diskSize"`

	// Source indicates the source of the assets to serve, in the form `gs://bucket-name/path`
	Source string `json:"source"`

	// MinReplicas indicates the minimal number of instances to deploy
	MinReplicas int32 `json:"minReplicas"`

	// MaxReplicas indicates the maximal number of instances to deploy
	MaxReplicas int32 `json:"maxReplicas"`
}

// StaticStatus defines the observed state of Static
type StaticStatus struct {
	// EXternalIP is the external IP of the load balancer
	ExternalIP string `json:"externalIP"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Static is the Schema for the statics API
type Static struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StaticSpec   `json:"spec,omitempty"`
	Status StaticStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// StaticList contains a list of Static
type StaticList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Static `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Static{}, &StaticList{})
}
