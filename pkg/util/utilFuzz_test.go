/* Copyright 2022 Red Hat, Inc.

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

package util

import (
	"errors"
	"regexp"
	"strings"
	"testing"
)

// run this test locally using go test -fuzz={FuzzTestName} in the test directory

func FuzzSanitizeErrorMessage(f *testing.F) {
	testcases := []string{"https://@github.com/fake/repo", "https://ghp_fj3492danj924@github.com/fake/repo", "ghp_A8jk2jsofle@github.com", "ghu_islaj29falkjsdf@github.com", "29IwharlkP1234fjiso@github.com"}
	for _, tc := range testcases {
		f.Add(tc) // Use f.Add to provide a seed corpus
	}
	reg, err := regexp.Compile(tokenRegex)
	f.Fuzz(func(t *testing.T, errMsg string) {
		//sanitize message
		sanitizedMsg := SanitizeErrorMessage(errors.New(errMsg))
		//verify message has been sanitized
		if err == nil && reg.MatchString(errMsg) {
			if !strings.Contains(sanitizedMsg.Error(), "<TOKEN>") {
				t.Errorf("String was not sanitized %q ", sanitizedMsg)
			}
		}
	})
}
