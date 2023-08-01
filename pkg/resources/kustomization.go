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

// From https://github.com/redhat-developer/kam/tree/master/pkg/pipelines/resources

package resources

import (
	"sort"
)

// Kustomization is a structural representation of the Kustomize file format.
type Kustomization struct {
	APIVersion   string            `json:"apiVersion,omitempty"`
	Kind         string            `json:"kind,omitempty"`
	Resources    []string          `json:"resources,omitempty"`
	Bases        []string          `json:"bases,omitempty"`
	Patches      []Patch           `json:"patches,omitempty"`
	CommonLabels map[string]string `json:"commonLabels,omitempty"`
}

// Patch holds the patch information
type Patch struct {
	Path string `json:"path"`
}

func (k *Kustomization) AddResources(s ...string) {
	k.Resources = removeDuplicatesAndSort(append(k.Resources, s...))
}

func (k *Kustomization) AddBases(s ...string) {
	k.Bases = removeDuplicatesAndSort(append(k.Bases, s...))
}

func (k *Kustomization) AddPatches(s ...string) {
	files := removeDuplicatesAndSort(append(getPatchFiles(k.Patches), s...))
	k.Patches = addFilestoPatches(files)
}

func removeDuplicatesAndSort(s []string) []string {
	exists := make(map[string]bool)
	out := []string{}
	for _, v := range s {
		if !exists[v] {
			out = append(out, v)
			exists[v] = true
		}
	}
	sort.Strings(out)
	return out
}

func (k *Kustomization) CompareDifferenceAndAddCustomPatches(original []Patch, generated []string) {
	newGeneratedFiles := []string{}
	originalPatches := make(map[string]bool)
	for _, originalElement := range original {
		originalPatches[originalElement.Path] = true
	}
	for _, generatedElement := range generated {
		if _, ok := originalPatches[generatedElement]; !ok {
			// preserve the newGeneratedFiles order
			newGeneratedFiles = append(newGeneratedFiles, generatedElement)
		}
	}
	// new generated files should add to the top of the patch list
	newPatchesList := append(newGeneratedFiles, getPatchFiles(original)...)
	k.Patches = addFilestoPatches(newPatchesList)
}

// gets the files from Patch
func getPatchFiles(patches []Patch) []string {
	var files []string
	for _, patch := range patches {
		files = append(files, patch.Path)
	}
	return files
}

// adds the files to Patch
func addFilestoPatches(files []string) []Patch {
	var patches []Patch
	for _, file := range files {
		patches = append(patches, Patch{Path: file})
	}
	return patches
}
