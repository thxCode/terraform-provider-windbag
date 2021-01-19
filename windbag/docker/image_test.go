package docker

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/stretchr/testify/assert"
)

func TestParseImage(t *testing.T) {
	// NB(thxCode): respect the Terraform Acceptance logic.
	if os.Getenv(resource.TestEnvVar) != "" {
		t.Skip(fmt.Sprintf(
			"Unit tests skipped as env '%s' set",
			resource.TestEnvVar))
		return
	}

	type input struct {
		image string
	}
	type output struct {
		image StructuredName
	}

	var testCases = []struct {
		name     string
		given    input
		expected output
	}{
		{
			name: "docker.io/library/ubuntu:21.04",
			given: input{
				image: "docker.io/library/ubuntu:21.04",
			},
			expected: output{
				image: StructuredName{
					Registry:   "docker.io",
					Repository: "library/ubuntu",
					Tag:        "21.04",
				},
			},
		},
		{
			name: "docker.io/library/ubuntu",
			given: input{
				image: "docker.io/library/ubuntu",
			},
			expected: output{
				image: StructuredName{
					Registry:   "docker.io",
					Repository: "library/ubuntu",
					Tag:        "latest",
				},
			},
		},
		{
			name: "library/ubuntu:20.10",
			given: input{
				image: "library/ubuntu:20.10",
			},
			expected: output{
				image: StructuredName{
					Registry:   "docker.io",
					Repository: "library/ubuntu",
					Tag:        "20.10",
				},
			},
		},
		{
			name: "ubuntu:latest",
			given: input{
				image: "ubuntu:latest",
			},
			expected: output{
				image: StructuredName{
					Registry:   "docker.io",
					Repository: "library/ubuntu",
					Tag:        "latest",
				},
			},
		},
		{
			name: "ubuntu",
			given: input{
				image: "ubuntu",
			},
			expected: output{
				image: StructuredName{
					Registry:   "docker.io",
					Repository: "library/ubuntu",
					Tag:        "latest",
				},
			},
		},
		{
			name: "thxcode/flannel-windows:v1.0.0",
			given: input{
				image: "thxcode/flannel-windows:v1.0.0",
			},
			expected: output{
				image: StructuredName{
					Registry:   "docker.io",
					Repository: "thxcode/flannel-windows",
					Tag:        "v1.0.0",
				},
			},
		},
		{
			name: "registry.cn-hangzhou.aliyuncs.com/acs/metrics-aggregator:v1.0.1",
			given: input{
				image: "registry.cn-hangzhou.aliyuncs.com/acs/metrics-aggregator:v1.0.1",
			},
			expected: output{
				image: StructuredName{
					Registry:   "registry.cn-hangzhou.aliyuncs.com",
					Repository: "acs/metrics-aggregator",
					Tag:        "v1.0.1",
				},
			},
		},
	}

	for _, tc := range testCases {
		var actual = ParseImage(tc.given.image)
		assert.Equal(t, tc.expected.image, actual, "case %q", tc.name)
	}
}

func TestGetImageDigest(t *testing.T) {
	// NB(thxCode): respect the Terraform Acceptance logic.
	if os.Getenv(resource.TestEnvVar) != "" {
		t.Skip(fmt.Sprintf(
			"Unit tests skipped as env '%s' set",
			resource.TestEnvVar))
		return
	}

	if os.Getenv("DOCKER_USERNAME") == "" {
		t.Skip("Skipped as the Docker login credential is not found")
		return
	}

	type input struct {
		image   string
		options []GetImageDigestOption
	}
	type output struct {
		digest string
		err    error
	}

	var testCases = []struct {
		name     string
		given    input
		expected output
	}{
		{
			name: "[WINDOWS] thxcode/logtail-windows:v1.0.10-1809",
			given: input{
				image: "thxcode/logtail-windows:v1.0.10-1809",
				options: []GetImageDigestOption{
					WithBasicAuth(os.Getenv("DOCKER_USERNAME"), os.Getenv("DOCKER_PASSWORD")),
					WithManifestSupport(),
				},
			},
			expected: output{
				digest: "sha256:ceda4fc4b5fe12950d951c73bc85a146f9c006ff1b28824d3a70e09064c0215c",
				err:    nil,
			},
		},
		{
			name: "[WINDOWS] mcr.microsoft.com/windows/servercore:1903-KB4592449",
			given: input{
				image: "mcr.microsoft.com/windows/servercore:1903-KB4592449",
				options: []GetImageDigestOption{
					WithBasicAuth(os.Getenv("DOCKER_USERNAME"), os.Getenv("DOCKER_PASSWORD")),
					WithManifestSupport(),
				},
			},
			expected: output{
				digest: "sha256:53076d8063287c3ec00e88598414968c01417db403ca3e0deeeaca7c2408ffe9",
				err:    nil,
			},
		},
	}

	for _, tc := range testCases {
		var actual output
		actual.digest, actual.err = GetImageDigest(context.Background(), tc.given.image, tc.given.options...)
		assert.Equal(t, actual, tc.expected, "case %q", tc.name)
	}
}
