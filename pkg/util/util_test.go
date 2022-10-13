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
	"testing"
)

func TestValidateRemoteURL(t *testing.T) {

	tests := []struct {
		name      string
		remoteURL string
		wantErr   error
	}{
		{
			name:      "Valid remote with gitlab domain",
			remoteURL: "https://2340908kjfas@gitlab.com/org/repo",
			wantErr:   nil,
		},
		{
			name:      "Invalid remote with unsupported domain",
			remoteURL: "https://2340908kjfas@xyz.com/org/repo",
			wantErr:   invalidRemoteMsg,
		},
		{
			name:      "Invalid remote with http scheme",
			remoteURL: "http://2340908kjfas@github.com/org/repo",
			wantErr:   invalidRemoteMsg,
		},
		{
			name:      "Valid remote with no token",
			remoteURL: "https://github.com/org/repo123.git",
			wantErr:   nil,
		},
		{
			name:      "Invalid remote with missing scheme",
			remoteURL: "/ghp_2340908kjfas@github.com/org/repo123/",
			wantErr:   invalidRemoteMsg,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRemote(tt.remoteURL)
			if err != tt.wantErr {
				t.Errorf("ValidateRemote() error: expected %v got %v", tt.wantErr, err)
			}
		})
	}
}
