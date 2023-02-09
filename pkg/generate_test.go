//
// Copyright 2021-2022 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gitops

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/redhat-developer/gitops-generator/pkg/testutils"
	"github.com/stretchr/testify/assert"

	routev1 "github.com/openshift/api/route/v1"
	gitopsv1alpha1 "github.com/redhat-developer/gitops-generator/api/v1alpha1"
	"github.com/redhat-developer/gitops-generator/pkg/resources"
	"github.com/redhat-developer/gitops-generator/pkg/util/ioutils"
	"github.com/spf13/afero"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"sigs.k8s.io/yaml"
)

func TestGenerateDeployment(t *testing.T) {
	applicationName := "test-application"
	componentName := "test-component"
	namespace := "test-namespace"
	replicas := int32(1)
	otherReplicas := int32(3)
	customK8slabels := map[string]string{
		"app.kubernetes.io/name":       componentName,
		"app.kubernetes.io/instance":   "ComponentCRName",
		"app.kubernetes.io/part-of":    applicationName,
		"app.kubernetes.io/managed-by": "kustomize",
		"app.kubernetes.io/created-by": "GitOps Generator Test",
	}
	k8slabels := map[string]string{
		"app.kubernetes.io/name":       componentName,
		"app.kubernetes.io/instance":   componentName,
		"app.kubernetes.io/part-of":    applicationName,
		"app.kubernetes.io/managed-by": "kustomize",
		"app.kubernetes.io/created-by": "application-service",
	}
	matchLabels := map[string]string{
		"app.kubernetes.io/instance": componentName,
	}

	tests := []struct {
		name           string
		component      gitopsv1alpha1.GeneratorOptions
		wantDeployment appsv1.Deployment
	}{
		{
			name: "Simple component, no optional fields set",
			component: gitopsv1alpha1.GeneratorOptions{
				Name:        componentName,
				Namespace:   namespace,
				Application: applicationName,
			},
			wantDeployment: appsv1.Deployment{
				TypeMeta: v1.TypeMeta{
					Kind:       "Deployment",
					APIVersion: "apps/v1",
				},
				ObjectMeta: v1.ObjectMeta{
					Name:      componentName,
					Namespace: namespace,
					Labels:    k8slabels,
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: &replicas,
					Selector: &v1.LabelSelector{
						MatchLabels: matchLabels,
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: v1.ObjectMeta{
							Labels: matchLabels,
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:            "container-image",
									ImagePullPolicy: corev1.PullAlways,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Component, optional fields set",
			component: gitopsv1alpha1.GeneratorOptions{
				Name:           componentName,
				Namespace:      namespace,
				Application:    applicationName,
				Replicas:       3,
				TargetPort:     5000,
				ContainerImage: "quay.io/test/test-image:latest",
				K8sLabels:      customK8slabels,
				BaseEnvVar: []corev1.EnvVar{
					{
						Name:  "test",
						Value: "value",
					},
				},
				Resources: corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("2M"),
						corev1.ResourceMemory: resource.MustParse("1Gi"),
					},
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("1M"),
						corev1.ResourceMemory: resource.MustParse("256Mi"),
					},
				},
			},
			wantDeployment: appsv1.Deployment{
				TypeMeta: v1.TypeMeta{
					Kind:       "Deployment",
					APIVersion: "apps/v1",
				},
				ObjectMeta: v1.ObjectMeta{
					Name:      componentName,
					Namespace: namespace,
					Labels:    customK8slabels,
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: &otherReplicas,
					Selector: &v1.LabelSelector{
						MatchLabels: matchLabels,
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: v1.ObjectMeta{
							Labels: matchLabels,
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:            "container-image",
									Image:           "quay.io/test/test-image:latest",
									ImagePullPolicy: corev1.PullAlways,
									Env: []corev1.EnvVar{
										{
											Name:  "test",
											Value: "value",
										},
									},
									Ports: []corev1.ContainerPort{
										{
											ContainerPort: int32(5000),
										},
									},
									ReadinessProbe: &corev1.Probe{
										InitialDelaySeconds: 10,
										PeriodSeconds:       10,
										ProbeHandler: corev1.ProbeHandler{
											TCPSocket: &corev1.TCPSocketAction{
												Port: intstr.FromInt(5000),
											},
										},
									},
									LivenessProbe: &corev1.Probe{
										InitialDelaySeconds: 10,
										PeriodSeconds:       10,
										ProbeHandler: corev1.ProbeHandler{
											HTTPGet: &corev1.HTTPGetAction{
												Port: intstr.FromInt(5000),
												Path: "/",
											},
										},
									},
									Resources: corev1.ResourceRequirements{
										Limits: corev1.ResourceList{
											corev1.ResourceCPU:    resource.MustParse("2M"),
											corev1.ResourceMemory: resource.MustParse("1Gi"),
										},
										Requests: corev1.ResourceList{
											corev1.ResourceCPU:    resource.MustParse("1M"),
											corev1.ResourceMemory: resource.MustParse("256Mi"),
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Simple image component, no optional fields set",
			component: gitopsv1alpha1.GeneratorOptions{
				Name:           componentName,
				Namespace:      namespace,
				Application:    applicationName,
				ContainerImage: "quay.io/test/test:latest",
			},
			wantDeployment: appsv1.Deployment{
				TypeMeta: v1.TypeMeta{
					Kind:       "Deployment",
					APIVersion: "apps/v1",
				},
				ObjectMeta: v1.ObjectMeta{
					Name:      componentName,
					Namespace: namespace,
					Labels:    k8slabels,
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: &replicas,
					Selector: &v1.LabelSelector{
						MatchLabels: matchLabels,
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: v1.ObjectMeta{
							Labels: matchLabels,
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:            "container-image",
									Image:           "quay.io/test/test:latest",
									ImagePullPolicy: corev1.PullAlways,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Simple image component with pull secret set",
			component: gitopsv1alpha1.GeneratorOptions{
				Name:           componentName,
				Namespace:      namespace,
				Application:    applicationName,
				Secret:         "my-image-pull-secret",
				ContainerImage: "quay.io/test/test:latest",
			},
			wantDeployment: appsv1.Deployment{
				TypeMeta: v1.TypeMeta{
					Kind:       "Deployment",
					APIVersion: "apps/v1",
				},
				ObjectMeta: v1.ObjectMeta{
					Name:      componentName,
					Namespace: namespace,
					Labels:    k8slabels,
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: &replicas,
					Selector: &v1.LabelSelector{
						MatchLabels: matchLabels,
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: v1.ObjectMeta{
							Labels: matchLabels,
						},
						Spec: corev1.PodSpec{
							ImagePullSecrets: []corev1.LocalObjectReference{
								{
									Name: "my-image-pull-secret",
								},
							},
							Containers: []corev1.Container{
								{
									Name:            "container-image",
									Image:           "quay.io/test/test:latest",
									ImagePullPolicy: corev1.PullAlways,
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generatedDeployment := generateDeployment(tt.component)

			if !reflect.DeepEqual(*generatedDeployment, tt.wantDeployment) {
				t.Errorf("TestGenerateDeployment() error: expected %v got %v", tt.wantDeployment, generatedDeployment)
			}
		})
	}
}

func TestGenerateDeploymentPatch(t *testing.T) {
	componentName := "test-component"
	namespace := "test-namespace"
	containerName := "test-container"
	replicas := int32(1)
	image := "image"

	tests := []struct {
		name           string
		component      gitopsv1alpha1.GeneratorOptions
		containerName  string
		imageName      string
		namespace      string
		wantDeployment appsv1.Deployment
	}{
		{
			name: "Simple component, no optional fields set",
			component: gitopsv1alpha1.GeneratorOptions{
				Name:     componentName,
				Replicas: int(replicas),
				BaseEnvVar: []corev1.EnvVar{
					{
						Name:  "FOO",
						Value: "BAR",
					},
				},
				Resources: corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("1"),
					},
				},
				OverlayEnvVar: []corev1.EnvVar{
					{
						Name:  "FOO",
						Value: "BAR_ENV",
					},
					{
						Name:  "FOO2",
						Value: "BAR2_ENV",
					},
				},
			},
			namespace:     namespace,
			imageName:     image,
			containerName: containerName,
			wantDeployment: appsv1.Deployment{
				TypeMeta: v1.TypeMeta{
					Kind:       "Deployment",
					APIVersion: "apps/v1",
				},
				ObjectMeta: v1.ObjectMeta{
					Name:      componentName,
					Namespace: namespace,
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: &replicas,
					Selector: &v1.LabelSelector{},
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  containerName,
									Image: image,
									Env: []corev1.EnvVar{
										{
											Name:  "FOO",
											Value: "BAR",
										},
										{
											Name:  "FOO2",
											Value: "BAR2_ENV",
										},
									},
									Resources: corev1.ResourceRequirements{
										Limits: corev1.ResourceList{
											corev1.ResourceCPU: resource.MustParse("1"),
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generatedDeployment := generateDeploymentPatch(tt.component, tt.imageName, tt.containerName, tt.namespace)

			if !reflect.DeepEqual(*generatedDeployment, tt.wantDeployment) {
				t.Errorf("TestGenerateDeploymentPatch() error: expected %v got %v", tt.wantDeployment, *generatedDeployment)
			}
		})
	}
}

func TestGenerateService(t *testing.T) {
	applicationName := "test-application"
	componentName := "test-component"
	namespace := "test-namespace"
	customK8sLabels := map[string]string{
		"app.kubernetes.io/name":       componentName,
		"app.kubernetes.io/instance":   "ComponentCRName",
		"app.kubernetes.io/part-of":    applicationName,
		"app.kubernetes.io/managed-by": "kustomize",
		"app.kubernetes.io/created-by": "GitOps Generator Test",
	}
	k8slabels := map[string]string{
		"app.kubernetes.io/name":       componentName,
		"app.kubernetes.io/instance":   componentName,
		"app.kubernetes.io/part-of":    applicationName,
		"app.kubernetes.io/managed-by": "kustomize",
		"app.kubernetes.io/created-by": "application-service",
	}
	matchLabels := map[string]string{
		"app.kubernetes.io/instance": componentName,
	}

	tests := []struct {
		name        string
		component   gitopsv1alpha1.GeneratorOptions
		wantService corev1.Service
	}{
		{
			name: "Simple component object",
			component: gitopsv1alpha1.GeneratorOptions{
				Name:        componentName,
				Namespace:   namespace,
				Application: applicationName,
				TargetPort:  5000,
			},
			wantService: corev1.Service{
				TypeMeta: v1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Service",
				},
				ObjectMeta: v1.ObjectMeta{
					Name:      componentName,
					Namespace: namespace,
					Labels:    k8slabels,
				},
				Spec: corev1.ServiceSpec{
					Selector: matchLabels,
					Ports: []corev1.ServicePort{
						{
							Port:       int32(5000),
							TargetPort: intstr.FromInt(5000),
						},
					},
				},
			},
		},
		{
			name: "Simple component object with custom k8s labels",
			component: gitopsv1alpha1.GeneratorOptions{
				Name:        componentName,
				Namespace:   namespace,
				Application: applicationName,
				TargetPort:  5000,
				K8sLabels:   customK8sLabels,
			},
			wantService: corev1.Service{
				TypeMeta: v1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Service",
				},
				ObjectMeta: v1.ObjectMeta{
					Name:      componentName,
					Namespace: namespace,
					Labels:    customK8sLabels,
				},
				Spec: corev1.ServiceSpec{
					Selector: matchLabels,
					Ports: []corev1.ServicePort{
						{
							Port:       int32(5000),
							TargetPort: intstr.FromInt(5000),
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generatedService := generateService(tt.component)

			if !reflect.DeepEqual(*generatedService, tt.wantService) {
				t.Errorf("TestGenerateService() error: expected %v got %v", tt.wantService, generatedService)
			}
		})
	}
}

func TestGenerateRoute(t *testing.T) {
	applicationName := "test-application"
	componentName := "test-component"
	namespace := "test-namespace"
	customK8sLabels := map[string]string{
		"app.kubernetes.io/name":       componentName,
		"app.kubernetes.io/instance":   "ComponentCRName",
		"app.kubernetes.io/part-of":    applicationName,
		"app.kubernetes.io/managed-by": "kustomize",
		"app.kubernetes.io/created-by": "GitOps Generator Test",
	}
	k8slabels := map[string]string{
		"app.kubernetes.io/name":       componentName,
		"app.kubernetes.io/instance":   componentName,
		"app.kubernetes.io/part-of":    applicationName,
		"app.kubernetes.io/managed-by": "kustomize",
		"app.kubernetes.io/created-by": "application-service",
	}
	weight := int32(100)

	tests := []struct {
		name      string
		component gitopsv1alpha1.GeneratorOptions
		wantRoute routev1.Route
	}{
		{
			name: "Simple component object",
			component: gitopsv1alpha1.GeneratorOptions{
				Name:        componentName,
				Namespace:   namespace,
				Application: applicationName,
				TargetPort:  5000,
			},
			wantRoute: routev1.Route{
				TypeMeta: v1.TypeMeta{
					Kind:       "Route",
					APIVersion: "route.openshift.io/v1",
				},
				ObjectMeta: v1.ObjectMeta{
					Name:      componentName,
					Namespace: namespace,
					Labels:    k8slabels,
				},
				Spec: routev1.RouteSpec{
					Port: &routev1.RoutePort{
						TargetPort: intstr.FromInt(5000),
					},
					TLS: &routev1.TLSConfig{
						InsecureEdgeTerminationPolicy: routev1.InsecureEdgeTerminationPolicyRedirect,
						Termination:                   routev1.TLSTerminationEdge,
					},
					To: routev1.RouteTargetReference{
						Kind:   "Service",
						Name:   componentName,
						Weight: &weight,
					},
				},
			},
		},
		{
			name: "Component object with route/hostname and custom k8s labels set",
			component: gitopsv1alpha1.GeneratorOptions{
				Name:        componentName,
				Namespace:   namespace,
				Application: applicationName,
				TargetPort:  5000,
				K8sLabels:   customK8sLabels,
				Route:       "example.com",
			},
			wantRoute: routev1.Route{
				TypeMeta: v1.TypeMeta{
					Kind:       "Route",
					APIVersion: "route.openshift.io/v1",
				},
				ObjectMeta: v1.ObjectMeta{
					Name:      componentName,
					Namespace: namespace,
					Labels:    customK8sLabels,
				},
				Spec: routev1.RouteSpec{
					Host: "example.com",
					Port: &routev1.RoutePort{
						TargetPort: intstr.FromInt(5000),
					},
					TLS: &routev1.TLSConfig{
						InsecureEdgeTerminationPolicy: routev1.InsecureEdgeTerminationPolicyRedirect,
						Termination:                   routev1.TLSTerminationEdge,
					},
					To: routev1.RouteTargetReference{
						Kind:   "Service",
						Name:   componentName,
						Weight: &weight,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generatedRoute := generateRoute(tt.component)

			if !reflect.DeepEqual(*generatedRoute, tt.wantRoute) {
				t.Errorf("TestGenerateRoute() error: expected %v got %v", tt.wantRoute, generatedRoute)
			}
		})
	}
}

func TestGenerateOverlays(t *testing.T) {
	component := gitopsv1alpha1.GeneratorOptions{
		Name: "test-component",
	}
	imageName := "test-image"
	namespace := "test-namespace"
	containerName := "test-container"

	fs := ioutils.NewMemoryFilesystem()
	readOnlyFs := ioutils.NewReadOnlyFs()

	// Prepopulate the fs with components
	gitOpsFolder := "/tmp/gitops"
	fs.MkdirAll(gitOpsFolder, 0755)
	baseFolder := filepath.Join(gitOpsFolder, "../", "base")
	fs.MkdirAll(baseFolder, 0755)
	baseDeploymentFilePath := filepath.Join(baseFolder, "deployment.yaml")
	baseDeployment := appsv1.Deployment{
		TypeMeta: v1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "test-component",
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &v1.LabelSelector{},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  containerName,
							Image: imageName,
						},
					},
				},
			},
		},
	}

	bytes, err := yaml.Marshal(baseDeployment)
	if err != nil {
		t.Errorf("unexpected error when marshal the base deployment yaml %v", err)
	}
	err = fs.WriteFile(baseDeploymentFilePath, bytes, 0755)
	if err != nil {
		t.Errorf("unexpected error when writing to base deployment file: %v", err)
	}

	outputFolder := filepath.Join(gitOpsFolder, "overlays")
	fs.MkdirAll(outputFolder, 0755)

	outputFolderWithKustomizationFile := filepath.Join(gitOpsFolder, "overlays-2")
	fs.MkdirAll(outputFolderWithKustomizationFile, 0755)
	preExistKustomizationFilepath := filepath.Join(outputFolderWithKustomizationFile, "kustomization.yaml")
	k := resources.Kustomization{
		Patches: []string{"patch1.yaml", "custom-patch1.yaml"},
	}
	bytes, err = yaml.Marshal(k)
	if err != nil {
		t.Errorf("unexpected error when marshal the kustomization yaml %v", err)
	}
	err = fs.WriteFile(preExistKustomizationFilepath, bytes, 0755)
	if err != nil {
		t.Errorf("unexpected error when writing to kustomizatipn file: %v", err)
	}

	invalidKustomizationFileFolder := filepath.Join(gitOpsFolder, "overlays-error")
	fs.MkdirAll(invalidKustomizationFileFolder, 0755)
	invalidKustomizationFilepath := filepath.Join(invalidKustomizationFileFolder, "kustomization.yaml")
	invalidKustomization := map[string]interface{}{
		"Resources": 8,
	}
	bytes, err = yaml.Marshal(invalidKustomization)
	if err != nil {
		t.Errorf("unexpected error when marshal the kustomization yaml %v", err)
	}
	err = fs.WriteFile(invalidKustomizationFilepath, bytes, 0755)
	if err != nil {
		t.Errorf("unexpected error when writing to kustomizatipn file: %v", err)
	}

	tests := []struct {
		name                        string
		fs                          afero.Afero
		outputFolder                string
		expectPatchEntries          int
		componentGeneratedResources map[string][]string
		wantErr                     string
	}{
		{
			name:               "simple success case",
			fs:                 fs,
			outputFolder:       outputFolder,
			expectPatchEntries: 1,
			wantErr:            "",
		},
		{
			name:               "existing kustomization file with custom patches",
			fs:                 fs,
			outputFolder:       outputFolderWithKustomizationFile,
			expectPatchEntries: 3,
			wantErr:            "",
		},
		{
			name:         "read only fs",
			fs:           readOnlyFs,
			outputFolder: outputFolderWithKustomizationFile,
			wantErr:      "failed to MkDirAll",
		},
		{
			name:         "unmarshall error",
			fs:           fs,
			outputFolder: invalidKustomizationFileFolder,
			wantErr:      " failed to unmarshal data: error unmarshaling JSON: while decoding JSON: json: cannot unmarshal number into Go struct field Kustomization.resources",
		},
		{
			name:         "genereated an additional patch",
			fs:           fs,
			outputFolder: outputFolderWithKustomizationFile,
			componentGeneratedResources: map[string][]string{
				"test-component": {
					"patch1.yaml",
				},
			},
			expectPatchEntries: 3,
			wantErr:            "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := GenerateOverlays(tt.fs, gitOpsFolder, tt.outputFolder, component, imageName, namespace, tt.componentGeneratedResources)

			if !testutils.ErrorMatch(t, tt.wantErr, err) {
				t.Errorf("unexpected error return value. Got %v", err)
			}

			if tt.wantErr == "" {
				// Validate that the deployment.yaml preserve the container name
				deploymentPatchFilepath := filepath.Join(tt.outputFolder, "deployment-patch.yaml")
				exists, err := tt.fs.Exists(deploymentPatchFilepath)
				if err != nil {
					t.Errorf("unexpected error checking if deployment file exists %v", err)
				}
				if !exists {
					t.Errorf("deployment file does not exist at path %v", deploymentPatchFilepath)
				}

				deployPatch := appsv1.Deployment{}
				deploymentPatchBytes, err := tt.fs.ReadFile(deploymentPatchFilepath)
				if err != nil {
					t.Errorf("unexpected error reading deployment file")
				}
				yaml.Unmarshal(deploymentPatchBytes, &deployPatch)
				if deployPatch.Spec.Template.Spec.Containers[0].Name != containerName {
					t.Errorf("expected container name %v, got %v", containerName, deployPatch.Spec.Template.Spec.Containers[0].Name)
				}

				// Validate that the kustomization.yaml got created successfully and contains the proper entries
				kustomizationFilepath := filepath.Join(tt.outputFolder, "kustomization.yaml")
				exists, err = tt.fs.Exists(kustomizationFilepath)
				if err != nil {
					t.Errorf("unexpected error checking if kustomize file exists %v", err)
				}
				if !exists {
					t.Errorf("kustomize file does not exist at path %v", kustomizationFilepath)
				}

				// Read the kustomization.yaml and validate its entries
				k := resources.Kustomization{}
				kustomizationBytes, err := tt.fs.ReadFile(kustomizationFilepath)
				if err != nil {
					t.Errorf("unexpected error reading parent kustomize file")
				}
				yaml.Unmarshal(kustomizationBytes, &k)

				// There match patch entries in the kustomization file
				if len(k.Patches) != tt.expectPatchEntries {
					t.Errorf("expected %v kustomization bases, got %v patches: %v", tt.expectPatchEntries, len(k.Patches), k.Patches)
				}

				// Validate that the APIVersion and Kind are set properly
				if k.Kind != "Kustomization" {
					t.Errorf("expected kustomize kind %v, got %v", "Kustomization", k.Kind)
				}
				if k.APIVersion != "kustomize.config.k8s.io/v1beta1" {
					t.Errorf("expected kustomize apiversion %v, got %v", "kustomize.config.k8s.io/v1beta1", k.APIVersion)
				}

			}
		})
	}
}

func TestGenerate(t *testing.T) {

	applicationName := "test-application"
	componentName := "test-component"
	namespace := "test-namespace"

	deployment1 := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "deployment1",
		},
	}
	deployment2 := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "deployment2",
		},
	}

	service1 := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "service1",
		},
	}
	service2 := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "service2",
		},
	}

	route1 := routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name: "route1",
		},
	}
	route2 := routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name: "route2",
		},
	}

	ingress1 := networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ingress1",
		},
	}
	ingress2 := networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ingress2",
		},
	}

	pod1 := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "pod1",
		},
	}

	others1 := []interface{}{
		deployment2,
		service2,
		route2,
	}

	others2 := []interface{}{
		pod1,
		deployment2,
		ingress1,
		ingress2,
	}

	fs := ioutils.NewFilesystem()

	tests := []struct {
		name                  string
		fs                    afero.Afero
		component             gitopsv1alpha1.GeneratorOptions
		outputFolder          string
		isDeploymentGenerated bool
		isServicetGenerated   bool
		isRouteGenerated      bool
		isSerializeRequired   bool // set to true if you are going to test KubernetesResources.Others
		wantFiles             map[string]interface{}
		wantErr               bool
	}{
		{
			name: "Single deployment object provided only",
			fs:   fs,
			component: gitopsv1alpha1.GeneratorOptions{
				Name:        componentName,
				Namespace:   namespace,
				Application: applicationName,
				KubernetesResources: gitopsv1alpha1.KubernetesResources{
					Deployments: []appsv1.Deployment{
						deployment1,
					},
				},
			},
			wantFiles: map[string]interface{}{
				kustomizeFileName: resources.Kustomization{
					APIVersion: "kustomize.config.k8s.io/v1beta1",
					Kind:       "Kustomization",
					Resources:  []string{deploymentFileName},
				},
				deploymentFileName: deployment1,
			},
		},
		{
			name: "Single svc object provided only",
			fs:   fs,
			component: gitopsv1alpha1.GeneratorOptions{
				Name:        componentName,
				Namespace:   namespace,
				Application: applicationName,
				KubernetesResources: gitopsv1alpha1.KubernetesResources{
					Services: []corev1.Service{
						service1,
					},
				},
			},
			isDeploymentGenerated: true,
			wantFiles: map[string]interface{}{
				kustomizeFileName: resources.Kustomization{
					APIVersion: "kustomize.config.k8s.io/v1beta1",
					Kind:       "Kustomization",
					Resources:  []string{deploymentFileName, serviceFileName},
				},
				serviceFileName: service1,
			},
		},
		{
			name: "Single route object provided only",
			fs:   fs,
			component: gitopsv1alpha1.GeneratorOptions{
				Name:        componentName,
				Namespace:   namespace,
				Application: applicationName,
				KubernetesResources: gitopsv1alpha1.KubernetesResources{
					Routes: []routev1.Route{
						route1,
					},
				},
			},
			isDeploymentGenerated: true,
			wantFiles: map[string]interface{}{
				kustomizeFileName: resources.Kustomization{
					APIVersion: "kustomize.config.k8s.io/v1beta1",
					Kind:       "Kustomization",
					Resources:  []string{deploymentFileName, routeFileName},
				},
				"route.yaml": route1,
			},
		},
		{
			name: "Single deployment object provided only, with Target Port should generate svc and route too",
			fs:   fs,
			component: gitopsv1alpha1.GeneratorOptions{
				Name:        componentName,
				Namespace:   namespace,
				Application: applicationName,
				KubernetesResources: gitopsv1alpha1.KubernetesResources{
					Deployments: []appsv1.Deployment{
						deployment1,
					},
				},
				TargetPort: 1234,
			},
			isServicetGenerated: true,
			isRouteGenerated:    true,
			wantFiles: map[string]interface{}{
				kustomizeFileName: resources.Kustomization{
					APIVersion: "kustomize.config.k8s.io/v1beta1",
					Kind:       "Kustomization",
					Resources:  []string{deploymentFileName, routeFileName, serviceFileName},
				},
				deploymentFileName: deployment1,
			},
		},
		{
			name: "Multiple deployment, service and route provided",
			fs:   fs,
			component: gitopsv1alpha1.GeneratorOptions{
				Name:        componentName,
				Namespace:   namespace,
				Application: applicationName,
				KubernetesResources: gitopsv1alpha1.KubernetesResources{
					Deployments: []appsv1.Deployment{
						deployment1,
						deployment2,
					},
					Services: []corev1.Service{
						service1,
						service2,
					},
					Routes: []routev1.Route{
						route1,
						route2,
					},
				},
				TargetPort: 1234,
			},
			isSerializeRequired: true,
			wantFiles: map[string]interface{}{
				kustomizeFileName: resources.Kustomization{
					APIVersion: "kustomize.config.k8s.io/v1beta1",
					Kind:       "Kustomization",
					Resources:  []string{deploymentFileName, otherFileName, routeFileName, serviceFileName},
				},
				deploymentFileName: deployment1,
				serviceFileName:    service1,
				routeFileName:      route1,
				otherFileName:      others1,
			},
		},
		{
			name: "Multiple deployments, ingresses and other multiple resources object provided only",
			fs:   fs,
			component: gitopsv1alpha1.GeneratorOptions{
				Name:        componentName,
				Namespace:   namespace,
				Application: applicationName,
				KubernetesResources: gitopsv1alpha1.KubernetesResources{
					Deployments: []appsv1.Deployment{
						deployment1,
						deployment2,
					},
					Ingresses: []networkingv1.Ingress{
						ingress1,
						ingress2,
					},
					Others: []interface{}{
						pod1,
					},
				},
			},
			isSerializeRequired: true,
			wantFiles: map[string]interface{}{
				kustomizeFileName: resources.Kustomization{
					APIVersion: "kustomize.config.k8s.io/v1beta1",
					Kind:       "Kustomization",
					Resources:  []string{deploymentFileName, otherFileName},
				},
				deploymentFileName: deployment1,
				otherFileName:      others2,
			},
		},
		{
			name:         "Error case with an invalid output path",
			fs:           ioutils.NewReadOnlyFs(),
			outputFolder: "~~~",
			component: gitopsv1alpha1.GeneratorOptions{
				Name:        componentName,
				Namespace:   namespace,
				Application: applicationName,
				KubernetesResources: gitopsv1alpha1.KubernetesResources{
					Deployments: []appsv1.Deployment{
						deployment1,
					},
				},
			},
			wantFiles: map[string]interface{}{
				kustomizeFileName: resources.Kustomization{
					APIVersion: "kustomize.config.k8s.io/v1beta1",
					Kind:       "Kustomization",
					Resources:  []string{deploymentFileName},
				},
				deploymentFileName: deployment1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			var outputFolder string
			if tt.outputFolder == "" {
				path, cleanup := makeTempDir(t)
				defer cleanup()
				outputFolder = filepath.ToSlash(filepath.Join(path, "manifest", "gitops"))
			} else {
				outputFolder = tt.outputFolder
			}

			// if resources are generated, add the generated resources to the wantFiles list
			if tt.isDeploymentGenerated {
				tt.wantFiles[deploymentFileName] = generateDeployment(tt.component)
			}

			if tt.isServicetGenerated {
				tt.wantFiles[serviceFileName] = generateService(tt.component)
			}

			if tt.isRouteGenerated {
				tt.wantFiles[routeFileName] = generateRoute(tt.component)
			}

			// serialize array interface to match file contents
			if tt.isSerializeRequired {
				separator := []byte("---\n")
				var data []byte
				notSerialized := tt.wantFiles[otherFileName]
				if v, ok := notSerialized.([]interface{}); ok {
					for _, o := range v {
						nestedData, err := yaml.Marshal(o)
						assertNoError(t, err)
						nestedData = append(nestedData, separator...)
						data = append(data, nestedData...)
					}
				}
				tt.wantFiles[otherFileName] = data
			}

			err := Generate(tt.fs, "", outputFolder, tt.component)
			if tt.wantErr && (err == nil) {
				t.Error("wanted error but got nil")
			} else if !tt.wantErr && err != nil {
				t.Errorf("got unexpected error: %v", err)
			} else if err == nil {
				assertResourcesExists(t, outputFolder, tt.wantFiles)
			}
		})
	}
}

func makeTempDir(t *testing.T) (string, func()) {
	t.Helper()
	dir, err := ioutil.TempDir(os.TempDir(), "manifest")
	assertNoError(t, err)
	return dir, func() {
		err := os.RemoveAll(dir)
		assertNoError(t, err)
	}
}

func assertResourcesExists(t *testing.T, outputFolder string, wantFiles map[string]interface{}) {

	t.Helper()

	fileInfos, err := ioutil.ReadDir(outputFolder)
	assertNoError(t, err)

	var generatedFiles []string
	for _, fi := range fileInfos {
		if !fi.IsDir() {
			generatedFiles = append(generatedFiles, fi.Name())
		}
	}

	for _, generatedFile := range generatedFiles {
		isExpectedFile := false
		for wantFileName, wantResource := range wantFiles {
			if generatedFile == wantFileName {
				isExpectedFile = true
				var want []byte
				if wantFileName != otherFileName {
					want, err = yaml.Marshal(wantResource)
					assertNoError(t, err)
				} else {
					if r, ok := wantResource.([]byte); ok {
						want = r
					} else {
						t.Fatalf("error reading wanted file %s", otherFileName)
					}
				}

				got, err := ioutil.ReadFile(filepath.Join(outputFolder, wantFileName))
				assertNoError(t, err)
				assert.Equal(t, want, got, "file %s should be equal", wantFileName)
			}
		}

		if isExpectedFile {
			delete(wantFiles, generatedFile)
		} else {
			t.Fatalf("file generated %s not expected", generatedFile)
		}
	}
}

// AssertNoError fails if there's an error
func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
