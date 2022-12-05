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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gitopsv1alpha1 "github.com/redhat-developer/gitops-generator/api/v1alpha1"
	"github.com/redhat-developer/gitops-generator/pkg/testutils"
	"github.com/redhat-developer/gitops-generator/pkg/util"
	"github.com/redhat-developer/gitops-generator/pkg/util/ioutils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

var originalExecute = execute

func TestCloneGenerateAndPush(t *testing.T) {
	repo := "https://github.com/testing/testing.git"
	repoWithToken := "https://ghu_28lafsjdifouwej@github.com/testing/testing.git"
	outputPath := "/fake/path"
	repoPath := "/fake/path/test-component"
	componentName := "test-component"
	component := gitopsv1alpha1.GeneratorOptions{
		ContainerImage: "testimage:latest",
		GitSource: &gitopsv1alpha1.GitSource{
			URL: repo,
		},
		TargetPort: 5000,
	}
	component.Name = "test-component"
	fs := ioutils.NewMemoryFilesystem()
	readOnlyFs := ioutils.NewReadOnlyFs()
	generator := NewGitopsGen()

	tests := []struct {
		name          string
		repo          string
		fs            afero.Afero
		component     gitopsv1alpha1.GeneratorOptions
		errors        *testutils.ErrorStack
		outputs       [][]byte
		want          []testutils.Execution
		wantErrString string
	}{
		{
			name:      "No errors",
			repo:      repo,
			fs:        fs,
			component: component,
			errors:    &testutils.ErrorStack{},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
				[]byte("test output4"),
				[]byte("test output5"),
				[]byte("test output6"),
				[]byte("test output7"),
			},
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, component.Name},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "rm",
					Args:    []string{"-rf", filepath.Join("components", componentName, "base")},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"add", "."},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"--no-pager", "diff", "--cached"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"commit", "-m", fmt.Sprintf("Generate GitOps base resources for component %s", componentName)},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"push", "origin", "main"},
				},
			},
		},
		{
			name:      "Git clone failure",
			repo:      repo,
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					nil,
					errors.New("test error"),
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
			},
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, component.Name},
				},
			},
			wantErrString: "test error",
		},
		{
			name:      "Git switch failure, git checkout failure",
			repo:      repo,
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					errors.New("Permission denied"),
					errors.New("Fatal error"),
					nil,
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
			},
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, component.Name},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"checkout", "-b", "main"},
				},
			},
			wantErrString: "failed to checkout branch \"main\" in repository \"/fake/path/test-component\" \"test output1\": Permission denied",
		},
		{
			name:      "Git switch failure, git checkout success",
			repo:      repo,
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					nil,
					errors.New("test error"),
					nil,
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
				[]byte("test output4"),
				[]byte("test output5"),
				[]byte("test output6"),
				[]byte("test output7"),
				[]byte("test output8"),
			},
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, component.Name},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"checkout", "-b", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "rm",
					Args:    []string{"-rf", filepath.Join("components", componentName, "base")},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"add", "."},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"--no-pager", "diff", "--cached"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"commit", "-m", fmt.Sprintf("Generate GitOps base resources for component %s", componentName)},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"push", "origin", "main"},
				},
			},
			wantErrString: "",
		},
		{
			name:      "rm -rf failure",
			repo:      repo,
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					errors.New("Permission Denied"),
					nil,
					nil,
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
			},
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, component.Name},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "rm",
					Args:    []string{"-rf", "components/test-component/base"},
				},
			},
			wantErrString: "failed to delete \"components/test-component/base\" folder in repository in \"/fake/path/test-component\" \"test output1\": Permission Denied",
		},
		{
			name:      "git add failure",
			repo:      repo,
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					errors.New("Fatal error"),
					nil,
					nil,
					nil,
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
				[]byte("test output4"),
			},
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, component.Name},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "rm",
					Args:    []string{"-rf", "components/test-component/base"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"add", "."},
				},
			},
			wantErrString: "failed to add files for component \"test-component\" to repository in \"/fake/path/test-component\" \"test output1\": Fatal error",
		},
		{
			name:      "git diff failure",
			repo:      repo,
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					errors.New("Permission Denied"),
					nil,
					nil,
					nil,
					nil,
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
				[]byte("test output4"),
				[]byte("test output5"),
			},
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, component.Name},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "rm",
					Args:    []string{"-rf", "components/test-component/base"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"add", "."},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"--no-pager", "diff", "--cached"},
				},
			},
			wantErrString: "failed to check git diff in repository \"/fake/path/test-component\" \"test output1\": Permission Denied",
		},
		{
			name:      "git commit failure",
			repo:      repo,
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					errors.New("Fatal error"),
					nil,
					nil,
					nil,
					nil,
					nil,
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
				[]byte("test output4"),
				[]byte("test output5"),
				[]byte("test output6"),
			},
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, component.Name},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "rm",
					Args:    []string{"-rf", "components/test-component/base"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"add", "."},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"--no-pager", "diff", "--cached"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"commit", "-m", fmt.Sprintf("Generate GitOps base resources for component %s", componentName)},
				},
			},
			wantErrString: "failed to commit files to repository \"/fake/path/test-component\" \"test output1\": Fatal error",
		},
		{
			name:      "git push failure with sanitized error message",
			repo:      repoWithToken,
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					errors.New("Fatal error"),
					nil,
					nil,
					nil,
					nil,
					nil,
					nil,
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
				[]byte("test output4"),
				[]byte("test output5"),
				[]byte("test output6"),
				[]byte("test output7"),
			},
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repoWithToken, component.Name},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "rm",
					Args:    []string{"-rf", "components/test-component/base"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"add", "."},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"--no-pager", "diff", "--cached"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"commit", "-m", fmt.Sprintf("Generate GitOps base resources for component %s", componentName)},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"push", "origin", "main"},
				},
			},
			wantErrString: util.SanitizeErrorMessage(fmt.Errorf("failed to push remote to repository \"%s\" \"test output1\": Fatal error", repoWithToken)).Error(),
		},
		{
			name:      "gitops generate failure",
			repo:      repo,
			fs:        readOnlyFs,
			component: component,
			errors:    &testutils.ErrorStack{},
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, component.Name},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "rm",
					Args:    []string{"-rf", "components/test-component/base"},
				},
			},
			wantErrString: "failed to generate the gitops resources in \"/fake/path/test-component/components/test-component/base\" for component \"test-component\"",
		},
		{
			name: "gitops generate failure - image component",
			repo: repo,
			fs:   readOnlyFs,
			component: gitopsv1alpha1.GeneratorOptions{
				Name:           "test-component",
				ContainerImage: "quay.io/test/test",
				TargetPort:     5000,
			},
			errors: &testutils.ErrorStack{},
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, "test-component"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "rm",
					Args:    []string{"-rf", "components/test-component/base"},
				},
			},
			wantErrString: "failed to generate the gitops resources in \"/fake/path/test-component/components/test-component/base\" for component \"test-component\": failed to MkDirAll",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			outputStack := testutils.NewOutputs(tt.outputs...)
			executedCmds := []testutils.Execution{}

			execute = newTestExecute(outputStack, tt.errors, &executedCmds)

			err := generator.CloneGenerateAndPush(outputPath, tt.repo, tt.component, tt.fs, "main", "/", true)

			if tt.wantErrString != "" {
				testutils.AssertErrorMatch(t, tt.wantErrString, err)
			} else {
				testutils.AssertNoError(t, err)
			}

			assert.Equal(t, tt.want, executedCmds, "command executed should be equal")
		})
	}

	execute = originalExecute

}

func TestGenerateOverlaysAndPush(t *testing.T) {
	repo := "https://github.com/testing/testing.git"
	outputPath := "/fake/path"
	repoPath := "/fake/path/test-application"
	componentName := "test-component"
	applicationName := "test-application"
	environmentName := "environment"
	imageName := "image"
	namespace := "namespace"
	component := gitopsv1alpha1.GeneratorOptions{
		Name:     componentName,
		Replicas: 2,
	}
	component.Name = "test-component"
	fs := ioutils.NewMemoryFilesystem()
	readOnlyFs := ioutils.NewReadOnlyFs()
	generator := NewGitopsGen()
	tests := []struct {
		name            string
		fs              afero.Afero
		component       gitopsv1alpha1.GeneratorOptions
		errors          *testutils.ErrorStack
		outputs         [][]byte
		applicationName string
		environmentName string
		imageName       string
		namespace       string
		want            []testutils.Execution
		wantErrString   string
	}{
		{
			name:      "No errors",
			fs:        fs,
			component: component,
			errors:    &testutils.ErrorStack{},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
				[]byte("test output4"),
				[]byte("test output5"),
				[]byte("test output6"),
			},
			applicationName: applicationName,
			environmentName: environmentName,
			imageName:       imageName,
			namespace:       namespace,
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, applicationName},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"add", "."},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"--no-pager", "diff", "--cached"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"commit", "-m", fmt.Sprintf("Generate %s environment overlays for component %s", environmentName, componentName)},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"push", "origin", "main"},
				},
			},
		},
		{
			name:      "Git clone failure",
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					nil,
					errors.New("test error"),
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
			},
			applicationName: applicationName,
			environmentName: environmentName,
			imageName:       imageName,
			namespace:       namespace,
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, applicationName},
				},
			},
			wantErrString: "test error",
		},
		{
			name:      "Git switch failure, git checkout failure",
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					errors.New("Permission denied"),
					errors.New("Fatal error"),
					nil,
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
			},
			applicationName: applicationName,
			environmentName: environmentName,
			imageName:       imageName,
			namespace:       namespace,
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, applicationName},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"checkout", "-b", "main"},
				},
			},
			wantErrString: "failed to checkout branch \"main\" in repository \"/fake/path/test-application\" \"test output1\": Permission denied",
		},
		{
			name:      "Git switch failure, git checkout success",
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					nil,
					errors.New("test error"),
					nil,
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
				[]byte("test output4"),
				[]byte("test output5"),
				[]byte("test output6"),
				[]byte("test output7"),
			},
			applicationName: applicationName,
			environmentName: environmentName,
			imageName:       imageName,
			namespace:       namespace,
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, applicationName},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"checkout", "-b", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"add", "."},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"--no-pager", "diff", "--cached"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"commit", "-m", fmt.Sprintf("Generate %s environment overlays for component %s", environmentName, componentName)},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"push", "origin", "main"},
				},
			},
			wantErrString: "",
		},
		{
			name:      "git add failure",
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					errors.New("Fatal error"),
					nil,
					nil,
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
			},
			applicationName: applicationName,
			environmentName: environmentName,
			imageName:       imageName,
			namespace:       namespace,
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, applicationName},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"add", "."},
				},
			},
			wantErrString: "failed to add files for component \"test-component\" to repository in \"/fake/path/test-application\" \"test output1\": Fatal error",
		},
		{
			name:      "git diff failure",
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					errors.New("Permission Denied"),
					nil,
					nil,
					nil,
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
				[]byte("test output4"),
			},
			applicationName: applicationName,
			environmentName: environmentName,
			imageName:       imageName,
			namespace:       namespace,
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, applicationName},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"add", "."},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"--no-pager", "diff", "--cached"},
				},
			},
			wantErrString: "failed to check git diff in repository \"/fake/path/test-application\" \"test output1\": Permission Denied",
		},
		{
			name:      "git commit failure",
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					errors.New("Fatal error"),
					nil,
					nil,
					nil,
					nil,
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
				[]byte("test output4"),
				[]byte("test output5"),
			},
			applicationName: applicationName,
			environmentName: environmentName,
			imageName:       imageName,
			namespace:       namespace,
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, applicationName},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"add", "."},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"--no-pager", "diff", "--cached"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"commit", "-m", fmt.Sprintf("Generate %s environment overlays for component %s", environmentName, componentName)},
				},
			},
			wantErrString: "failed to commit files to repository \"/fake/path/test-application\" \"test output1\": Fatal error",
		},
		{
			name:      "git push failure",
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					errors.New("Fatal error"),
					nil,
					nil,
					nil,
					nil,
					nil,
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
				[]byte("test output4"),
				[]byte("test output5"),
				[]byte("test output6"),
			},
			applicationName: applicationName,
			environmentName: environmentName,
			imageName:       imageName,
			namespace:       namespace,
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, applicationName},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"add", "."},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"--no-pager", "diff", "--cached"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"commit", "-m", fmt.Sprintf("Generate %s environment overlays for component %s", environmentName, componentName)},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"push", "origin", "main"},
				},
			},
			wantErrString: fmt.Sprintf("failed to push remote to repository \"%s\" \"test output1\": Fatal error", repo),
		},
		{
			name:            "gitops generate failure",
			fs:              readOnlyFs,
			component:       component,
			errors:          &testutils.ErrorStack{},
			applicationName: applicationName,
			environmentName: environmentName,
			imageName:       imageName,
			namespace:       namespace,
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, applicationName},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
			},
			wantErrString: "failed to generate the gitops resources in overlays dir \"/fake/path/test-application/components/test-component/overlays/environment\" for component \"test-component\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generatedResources := make(map[string][]string)
			outputStack := testutils.NewOutputs(tt.outputs...)
			executedCmds := []testutils.Execution{}

			execute = newTestExecute(outputStack, tt.errors, &executedCmds)

			err := generator.GenerateOverlaysAndPush(outputPath, true, repo, tt.component, tt.applicationName, tt.environmentName, tt.imageName, tt.namespace, tt.fs, "main", "/", true, generatedResources)

			if tt.wantErrString != "" {
				testutils.AssertErrorMatch(t, tt.wantErrString, err)
			} else {
				testutils.AssertNoError(t, err)
				assert.Equal(t, 1, len(generatedResources), "should be equal")
				hasGitopsGeneratedResource := map[string]bool{
					"deployment-patch.yaml": true,
				}

				for _, generatedRes := range generatedResources[componentName] {
					assert.True(t, hasGitopsGeneratedResource[generatedRes], "should be equal")
				}
			}

			assert.Equal(t, tt.want, executedCmds, "command executed should be equal")
		})
	}
	execute = originalExecute
}

func TestGitRemoveComponent(t *testing.T) {
	repo := "https://github.com/testing/testing.git"
	outputPath := "/fake/path"
	repoPath := "/fake/path/test-component"
	componentPath := "/fake/path/test-component/components/test-component"
	componentBasePath := "/fake/path/test-component/components/test-component/base"
	componentName := "test-component"
	component := gitopsv1alpha1.GeneratorOptions{
		GitSource: &gitopsv1alpha1.GitSource{
			URL: repo,
		},
		TargetPort: 5000,
	}
	component.Name = "test-component"
	fs := ioutils.NewMemoryFilesystem()
	generator := NewGitopsGen()

	tests := []struct {
		name          string
		fs            afero.Afero
		component     gitopsv1alpha1.GeneratorOptions
		errors        *testutils.ErrorStack
		outputs       [][]byte
		want          []testutils.Execution
		wantErrString string
	}{
		{
			name:      "No errors",
			fs:        fs,
			component: component,
			errors:    &testutils.ErrorStack{},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
				[]byte("test output4"),
				[]byte("test output5"),
				[]byte("test output6"),
				[]byte("test output7"),
			},
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, component.Name},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "rm",
					Args:    []string{"-rf", componentPath},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"add", "."},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"--no-pager", "diff", "--cached"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"commit", "-m", fmt.Sprintf("Removed component %s", componentName)},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"push", "origin", "main"},
				},
			},
		},
		{
			name:      "Git clone failure",
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					nil,
					errors.New("test error"),
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
			},
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, component.Name},
				},
			},
			wantErrString: "test error",
		},
		{
			name:      "Git switch failure, git checkout failure",
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					errors.New("Permission denied"),
					errors.New("Fatal error"),
					nil,
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
			},
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, component.Name},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"checkout", "-b", "main"},
				},
			},
			wantErrString: "failed to checkout branch \"main\" in repository \"/fake/path/test-component\" \"test output1\": Permission denied",
		},
		{
			name:      "Git switch failure, git checkout success",
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					nil,
					errors.New("test error"),
					nil,
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
				[]byte("test output4"),
				[]byte("test output5"),
				[]byte("test output6"),
				[]byte("test output7"),
				[]byte("test output8"),
			},
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, component.Name},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"checkout", "-b", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "rm",
					Args:    []string{"-rf", componentPath},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"add", "."},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"--no-pager", "diff", "--cached"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"commit", "-m", fmt.Sprintf("Removed component %s", componentName)},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"push", "origin", "main"},
				},
			},
			wantErrString: "",
		},
		{
			name:      "rm -rf failure",
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					errors.New("Permission Denied"),
					nil,
					nil,
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
			},
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, component.Name},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "rm",
					Args:    []string{"-rf", componentPath},
				},
			},
			wantErrString: "failed to delete \"/fake/path/test-component/components/test-component\" folder in repository in \"/fake/path/test-component\" \"test output1\": Permission Denied",
		},
		{
			name:      "git add failure",
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					errors.New("Fatal error"),
					nil,
					nil,
					nil,
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
				[]byte("test output4"),
			},
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, component.Name},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "rm",
					Args:    []string{"-rf", componentPath},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"add", "."},
				},
			},
			wantErrString: "failed to add files for component \"test-component\" to repository in \"/fake/path/test-component\" \"test output1\": Fatal error",
		},
		{
			name:      "git diff failure",
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					errors.New("Permission Denied"),
					nil,
					nil,
					nil,
					nil,
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
				[]byte("test output4"),
				[]byte("test output5"),
			},
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, component.Name},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "rm",
					Args:    []string{"-rf", componentPath},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"add", "."},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"--no-pager", "diff", "--cached"},
				},
			},
			wantErrString: "failed to check git diff in repository \"/fake/path/test-component\" \"test output1\": Permission Denied",
		},
		{
			name:      "git commit failure",
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					errors.New("Fatal error"),
					nil,
					nil,
					nil,
					nil,
					nil,
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
				[]byte("test output4"),
				[]byte("test output5"),
				[]byte("test output6"),
			},
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, component.Name},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "rm",
					Args:    []string{"-rf", componentPath},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"add", "."},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"--no-pager", "diff", "--cached"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"commit", "-m", fmt.Sprintf("Removed component %s", componentName)},
				},
			},
			wantErrString: "failed to commit files to repository \"/fake/path/test-component\" \"test output1\": Fatal error",
		},
		{
			name:      "git push failure",
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					errors.New("Fatal error"),
					nil,
					nil,
					nil,
					nil,
					nil,
					nil,
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
				[]byte("test output4"),
				[]byte("test output5"),
				[]byte("test output6"),
				[]byte("test output7"),
			},
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, component.Name},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "rm",
					Args:    []string{"-rf", componentPath},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"add", "."},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"--no-pager", "diff", "--cached"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"commit", "-m", fmt.Sprintf("Removed component %s", componentName)},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"push", "origin", "main"},
				},
			},
			wantErrString: fmt.Sprintf("failed to push remote to repository \"%s\" \"test output1\": Fatal error", repo),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			outputStack := testutils.NewOutputs(tt.outputs...)
			executedCmds := []testutils.Execution{}

			execute = newTestExecute(outputStack, tt.errors, &executedCmds)

			if err := Generate(fs, repoPath, componentBasePath, tt.component); err != nil {
				t.Errorf("unexpected error %v", err)
				return
			}

			err := generator.GitRemoveComponent(outputPath, repo, tt.component.Name, "main", "/")

			if tt.wantErrString != "" {
				testutils.AssertErrorMatch(t, tt.wantErrString, err)
			} else {
				testutils.AssertNoError(t, err)
			}

			assert.Equal(t, tt.want, executedCmds, "command executed should be equal")
		})
	}

	execute = originalExecute
}

func TestRemoveComponent(t *testing.T) {
	repo := "https://github.com/testing/testing.git"
	outputPath := "/fake/path"
	repoPath := "/fake/path/test-component"
	componentPath := "/fake/path/test-component/components/test-component"
	componentBasePath := "/fake/path/test-component/components/test-component/base"
	componentName := "test-component"
	component := gitopsv1alpha1.GeneratorOptions{
		GitSource: &gitopsv1alpha1.GitSource{
			URL: repo,
		},
		TargetPort: 5000,
	}
	component.Name = "test-component"
	fs := ioutils.NewMemoryFilesystem()
	generator := NewGitopsGen()
	tests := []struct {
		name                string
		fs                  afero.Afero
		component           gitopsv1alpha1.GeneratorOptions
		errors              *testutils.ErrorStack
		outputs             [][]byte
		want                []testutils.Execution
		wantCloneErrString  string
		wantRemoveErrString string
		wantPushErrString   string
	}{
		{
			name:      "No errors",
			fs:        fs,
			component: component,
			errors:    &testutils.ErrorStack{},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
				[]byte("test output4"),
				[]byte("test output5"),
				[]byte("test output6"),
				[]byte("test output7"),
			},
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, component.Name},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "rm",
					Args:    []string{"-rf", componentPath},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"add", "."},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"--no-pager", "diff", "--cached"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"commit", "-m", fmt.Sprintf("Removed component %s", componentName)},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"push", "origin", "main"},
				},
			},
		},
		{
			name:      "Git clone failure",
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					nil,
					errors.New("test error"),
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
			},
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, component.Name},
				},
			},
			wantCloneErrString: "test error",
		},
		{
			name:      "Git switch failure, git checkout failure",
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					errors.New("Permission denied"),
					errors.New("Fatal error"),
					nil,
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
			},
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, component.Name},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"checkout", "-b", "main"},
				},
			},
			wantCloneErrString: "failed to checkout branch \"main\" in repository \"/fake/path/test-component\" \"test output1\": Permission denied",
		},
		{
			name:      "Git switch failure, git checkout success",
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					nil,
					errors.New("test error"),
					nil,
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
				[]byte("test output4"),
				[]byte("test output5"),
				[]byte("test output6"),
				[]byte("test output7"),
				[]byte("test output8"),
			},
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, component.Name},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"checkout", "-b", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "rm",
					Args:    []string{"-rf", componentPath},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"add", "."},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"--no-pager", "diff", "--cached"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"commit", "-m", fmt.Sprintf("Removed component %s", componentName)},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"push", "origin", "main"},
				},
			},
			wantCloneErrString: "",
		},
		{
			name:      "rm -rf failure",
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					errors.New("Permission Denied"),
					nil,
					nil,
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
			},
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, component.Name},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "rm",
					Args:    []string{"-rf", componentPath},
				},
			},
			wantRemoveErrString: "failed to delete \"/fake/path/test-component/components/test-component\" folder in repository in \"/fake/path/test-component\" \"test output1\": Permission Denied",
		},
		{
			name:      "git add failure",
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					errors.New("Fatal error"),
					nil,
					nil,
					nil,
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
				[]byte("test output4"),
			},
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, component.Name},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "rm",
					Args:    []string{"-rf", componentPath},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"add", "."},
				},
			},
			wantPushErrString: "failed to add files for component \"test-component\" to repository in \"/fake/path/test-component\" \"test output1\": Fatal error",
		},
		{
			name:      "git diff failure",
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					errors.New("Permission Denied"),
					nil,
					nil,
					nil,
					nil,
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
				[]byte("test output4"),
				[]byte("test output5"),
			},
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, component.Name},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "rm",
					Args:    []string{"-rf", componentPath},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"add", "."},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"--no-pager", "diff", "--cached"},
				},
			},
			wantPushErrString: "failed to check git diff in repository \"/fake/path/test-component\" \"test output1\": Permission Denied",
		},
		{
			name:      "git commit failure",
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					errors.New("Fatal error"),
					nil,
					nil,
					nil,
					nil,
					nil,
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
				[]byte("test output4"),
				[]byte("test output5"),
				[]byte("test output6"),
			},
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, component.Name},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "rm",
					Args:    []string{"-rf", componentPath},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"add", "."},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"--no-pager", "diff", "--cached"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"commit", "-m", fmt.Sprintf("Removed component %s", componentName)},
				},
			},
			wantPushErrString: "failed to commit files to repository \"/fake/path/test-component\" \"test output1\": Fatal error",
		},
		{
			name:      "git push failure",
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					errors.New("Fatal error"),
					nil,
					nil,
					nil,
					nil,
					nil,
					nil,
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
				[]byte("test output4"),
				[]byte("test output5"),
				[]byte("test output6"),
				[]byte("test output7"),
			},
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, component.Name},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "rm",
					Args:    []string{"-rf", componentPath},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"add", "."},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"--no-pager", "diff", "--cached"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"commit", "-m", fmt.Sprintf("Removed component %s", componentName)},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"push", "origin", "main"},
				},
			},
			wantPushErrString: fmt.Sprintf("failed to push remote to repository \"%s\" \"test output1\": Fatal error", repo),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputStack := testutils.NewOutputs(tt.outputs...)
			executedCmds := []testutils.Execution{}

			execute = newTestExecute(outputStack, tt.errors, &executedCmds)

			if err := Generate(fs, repoPath, componentBasePath, tt.component); err != nil {
				t.Errorf("unexpected error %v", err)
				return
			}

			err := generator.CloneRepo(outputPath, repo, tt.component.Name, "main")

			if tt.wantCloneErrString != "" {
				testutils.AssertErrorMatch(t, tt.wantCloneErrString, err)
			} else {
				testutils.AssertNoError(t, err)
			}

			if tt.wantCloneErrString == "" {

				err = removeComponent(outputPath, tt.component.Name, "/")

				if tt.wantRemoveErrString != "" {
					testutils.AssertErrorMatch(t, tt.wantRemoveErrString, err)
				} else {
					testutils.AssertNoError(t, err)
				}

				if tt.wantRemoveErrString == "" {

					err = generator.CommitAndPush(outputPath, "", repo, tt.component.Name, "main", fmt.Sprintf("Removed component %s", componentName))

					if tt.wantPushErrString != "" {
						testutils.AssertErrorMatch(t, tt.wantPushErrString, err)
					} else {
						testutils.AssertNoError(t, err)
					}

				}
			}

			assert.Equal(t, tt.want, executedCmds, "command executed should be equal")

		})
	}
	execute = originalExecute
}

func TestExecute(t *testing.T) {
	tests := []struct {
		name       string
		command    CommandType
		outputPath string
		args       string
		wantErr    error
	}{
		{
			name:    "Simple command to execute",
			command: GitCommand,
			args:    "help",
			wantErr: nil,
		},
		{
			name:    "Invalid command, error expected",
			command: "cd",
			args:    "/",
			wantErr: fmt.Errorf(unsupportedCmdMsg, "cd"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputStack := testutils.NewOutputs()
			executedCmds := []testutils.Execution{}

			execute = newTestExecute(outputStack, testutils.NewErrors(), &executedCmds)

			_, err := execute(tt.outputPath, tt.command, tt.args)

			if tt.wantErr != nil && err != nil {
				if tt.wantErr.Error() != err.Error() {
					t.Errorf("TestExecute() unexpected error: %v, want error: %v ", err, tt.wantErr)
				}
			}

			if tt.wantErr == nil && err != nil {
				t.Errorf("TestExecute() unexpected error: %v, want error: nil ", err)
			}

			if tt.wantErr != nil && err == nil {
				t.Errorf("TestExecute() expected want error: %v, got error: nil ", tt.wantErr)
			}
		})
	}
	execute = originalExecute
}

func TestGenerateAndPush(t *testing.T) {
	repo := "https://github.com/testing/testing.git"
	outputPath := "/fake/path"
	component := gitopsv1alpha1.GeneratorOptions{
		ContainerImage: "testimage:latest",
		GitSource:      &gitopsv1alpha1.GitSource{},
		TargetPort:     5000,
	}
	component.Name = "test-component"
	fs := ioutils.NewMemoryFilesystem()
	generator := NewGitopsGen()
	tests := []struct {
		name          string
		fs            afero.Afero
		component     gitopsv1alpha1.GeneratorOptions
		errors        *testutils.ErrorStack
		outputs       [][]byte
		doPush        bool
		repo          string
		want          []testutils.Execution
		wantErrString string
	}{
		{
			name:      "No errors. GenerateAndPush test with no push",
			fs:        fs,
			component: component,
			doPush:    false,
			repo:      "https://github.com/testing/testing.git",
			errors:    &testutils.ErrorStack{},
			want:      []testutils.Execution{},
		},
		{
			name:          "GenerateAndPush test with push.  Client access error",
			fs:            fs,
			component:     component,
			doPush:        true,
			repo:          "https://xyz/testing/testing.git",
			errors:        &testutils.ErrorStack{},
			want:          []testutils.Execution{},
			wantErrString: "failed to create a client to access \"https://xyz/testing/testing.git\": unable to identify driver from hostname: xyz",
		},
		{
			name:          "GenerateAndPush test with push.  Unauthorized user error",
			fs:            fs,
			component:     component,
			doPush:        true,
			repo:          "https://github.com/testing/testing.git",
			errors:        &testutils.ErrorStack{},
			want:          []testutils.Execution{},
			wantErrString: "failed to get the user with their auth token: Unauthorized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputStack := testutils.NewOutputs(tt.outputs...)
			executedCmds := []testutils.Execution{}
			component.GitSource.URL = tt.repo
			execute = newTestExecute(outputStack, tt.errors, &executedCmds)
			err := generator.GenerateAndPush(outputPath, repo, tt.component, tt.fs, "main", tt.doPush, "KAM CLI")

			if tt.wantErrString != "" {
				testutils.AssertErrorMatch(t, tt.wantErrString, err)
			} else {
				testutils.AssertNoError(t, err)
			}

			assert.Equal(t, tt.want, executedCmds, "command executed should be equal")
		})
	}
	execute = originalExecute
}

func TestGetCommitIDFromRepo(t *testing.T) {
	// Create an empty git repository and git commit to test with
	fs := ioutils.NewFilesystem()
	tempDir, err := fs.TempDir(os.TempDir(), "test")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	err = createEmptyGitRepository(tempDir)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	commitID, err := getCommitIDFromDotGit(tempDir)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	generator := NewGitopsGen()
	tests := []struct {
		name        string
		useMockExec bool
		repoPath    string
		want        string
		wantErr     bool
	}{
		{
			name:        "No errors, successfully retrieve git commit ID",
			useMockExec: false,
			repoPath:    tempDir,
			want:        commitID,
			wantErr:     false,
		},
		{
			name:        "Invalid git repo, no commit ID",
			useMockExec: false,
			repoPath:    os.TempDir(),
			want:        "",
			wantErr:     true,
		},
		{
			name:        "Test with mock executor, should pass",
			useMockExec: true,
			repoPath:    os.TempDir(),
			want:        "ca82a6dff817ec66f44342007202690a93763949",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if tt.useMockExec {
				outputStack := testutils.NewOutputs()
				executedCmds := []testutils.Execution{}

				execute = newTestExecute(outputStack, testutils.NewErrors(), &executedCmds)
			}

			commitID, err := generator.GetCommitIDFromRepo(fs, tt.repoPath)

			if err != nil && !tt.wantErr {
				t.Errorf("TestGetCommitIDFromRepo() unexpected error: %s", err.Error())
			}
			if err == nil && tt.wantErr {
				t.Errorf("TestGetCommitIDFromRepo() did not get expected error")
			}
			if commitID != tt.want {
				t.Errorf("TestGetCommitIDFromRepo() wanted: %v, got: %v", tt.want, commitID)
			}
		})
	}
	execute = originalExecute
}

// createEmptyGitRepository generates an empty git repository under the specified folder
func createEmptyGitRepository(repoPath string) error {
	// Initialize the Git repository
	if out, err := execute(repoPath, GitCommand, "init"); err != nil {
		return fmt.Errorf("Unable to intialize git repository in %q %q: %s", repoPath, out, err)
	}

	// Create an empty commit
	if out, err := execute(repoPath, GitCommand, "-c", "user.name='Test User'", "-c", "user.email='test@test.org'", "commit", "--allow-empty", "-m", "\"Empty commit\""); err != nil {
		return fmt.Errorf("Unable to create empty commit in %q %q: %s", repoPath, out, err)
	}
	return nil
}

// getCommitIDFromDotGit returns the latest commit ID for the default branch in the given git repository
func getCommitIDFromDotGit(repoPath string) (string, error) {
	fs := ioutils.NewFilesystem()
	var fileBytes []byte
	fileBytes, err := fs.ReadFile(filepath.Join(repoPath, ".git", "refs", "heads", "main"))
	if err != nil {
		return "", err
	}
	return string(fileBytes), nil
}

func mockExecute(outputStack *testutils.OutputStack, errorStack *testutils.ErrorStack, executedCmds *[]testutils.Execution, baseDir string, cmd CommandType, args ...string) ([]byte, error, *[]testutils.Execution) {
	if cmd == GitCommand || cmd == RmCommand {
		*executedCmds = append(*executedCmds, testutils.Execution{BaseDir: baseDir, Command: string(cmd), Args: args})
		if len(args) > 0 && args[0] == "rev-parse" {
			if strings.Contains(baseDir, "test-git-error") {
				return []byte(""), fmt.Errorf("unable to retrive git commit id"), executedCmds
			} else {
				return []byte("ca82a6dff817ec66f44342007202690a93763949"), errorStack.Pop(), executedCmds
			}
		} else {
			return outputStack.Pop(), errorStack.Pop(), executedCmds
		}
	}

	return []byte(""), fmt.Errorf("Unsupported command \"%s\" ", string(cmd)), executedCmds
}

func newTestExecute(outputStack *testutils.OutputStack, errorStack *testutils.ErrorStack, executedCmds *[]testutils.Execution) func(baseDir string, cmd CommandType, args ...string) ([]byte, error) {
	return func(baseDir string, cmd CommandType, args ...string) ([]byte, error) {
		var output []byte
		var execErr error
		output, execErr, executedCmds = mockExecute(outputStack, errorStack, executedCmds, baseDir, cmd, args...)
		return output, execErr
	}
}
