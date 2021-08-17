package docker

import (
	"bytes"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/stretchr/testify/assert"
	"io"
	"os"
	"testing"
)

func TestInjectTargetPlatformArgsToDockerfile(t *testing.T) {
	// NB(thxCode): respect the Terraform Acceptance logic.
	if os.Getenv(resource.TestEnvVar) != "" {
		t.Skip(fmt.Sprintf(
			"Unit tests skipped as env '%s' set",
			resource.TestEnvVar))
		return
	}

	type input struct {
		raw     io.Reader
		os      string
		arch    string
		variant string
	}
	type output struct {
		changed string
	}

	var testCases = []struct {
		name     string
		given    input
		expected output
	}{
		{
			name: "full",
			given: input{
				raw: bytes.NewBufferString(`
ARG RELEASEID=1809
# NB(thxCode): automatic platform ARGs, ref to:
# - https://docs.docker.com/engine/reference/builder/#automatic-platform-args-in-the-global-scope
ARG TARGETPLATFORM
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT

FROM mcr.microsoft.com/windows/servercore:${RELEASEID} as builder

ENTRYPOINT ["powershell.exe", "-NoLogo"]
`),
				os:      "windows",
				arch:    "amd64",
				variant: "1809",
			},
			expected: output{
				changed: `
ARG RELEASEID=1809
# NB(thxCode): automatic platform ARGs, ref to:
# - https://docs.docker.com/engine/reference/builder/#automatic-platform-args-in-the-global-scope
ARG TARGETPLATFORM="windows/amd64"
ARG TARGETOS="windows"
ARG TARGETARCH="amd64"
ARG TARGETVARIANT="1809"

FROM mcr.microsoft.com/windows/servercore:${RELEASEID} as builder

ENTRYPOINT ["powershell.exe", "-NoLogo"]
`,
			},
		},
		{
			name: "blank variant",
			given: input{
				raw: bytes.NewBufferString(`
ARG RELEASEID=1809
# NB(thxCode): automatic platform ARGs, ref to:
# - https://docs.docker.com/engine/reference/builder/#automatic-platform-args-in-the-global-scope
ARG TARGETPLATFORM
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT

FROM mcr.microsoft.com/windows/servercore:${RELEASEID} as builder

ENTRYPOINT ["powershell.exe", "-NoLogo"]
`),
				os:      "windows",
				arch:    "amd64",
				variant: "",
			},
			expected: output{
				changed: `
ARG RELEASEID=1809
# NB(thxCode): automatic platform ARGs, ref to:
# - https://docs.docker.com/engine/reference/builder/#automatic-platform-args-in-the-global-scope
ARG TARGETPLATFORM="windows/amd64"
ARG TARGETOS="windows"
ARG TARGETARCH="amd64"
ARG TARGETVARIANT=""

FROM mcr.microsoft.com/windows/servercore:${RELEASEID} as builder

ENTRYPOINT ["powershell.exe", "-NoLogo"]
`,
			},
		},
		{
			name: "lack of descriptor",
			given: input{
				raw: bytes.NewBufferString(`
ARG RELEASEID=1809
# NB(thxCode): automatic platform ARGs, ref to:
# - https://docs.docker.com/engine/reference/builder/#automatic-platform-args-in-the-global-scope
ARG TARGETOS

FROM mcr.microsoft.com/windows/servercore:${RELEASEID} as builder

ENTRYPOINT ["powershell.exe", "-NoLogo"]
`),
				os:      "windows",
				arch:    "1809",
				variant: "1809",
			},
			expected: output{
				changed: `
ARG RELEASEID=1809
# NB(thxCode): automatic platform ARGs, ref to:
# - https://docs.docker.com/engine/reference/builder/#automatic-platform-args-in-the-global-scope
ARG TARGETOS="windows"

FROM mcr.microsoft.com/windows/servercore:${RELEASEID} as builder

ENTRYPOINT ["powershell.exe", "-NoLogo"]
`,
			},
		},
		{
			name: "invalid os",
			given: input{
				raw: bytes.NewBufferString(`
ARG RELEASEID=1809
# NB(thxCode): automatic platform ARGs, ref to:
# - https://docs.docker.com/engine/reference/builder/#automatic-platform-args-in-the-global-scope
ARG TARGETPLATFORM
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT

FROM mcr.microsoft.com/windows/servercore:${RELEASEID} as builder

ENTRYPOINT ["powershell.exe", "-NoLogo"]
`),
				os:      "",
				arch:    "1809",
				variant: "1809",
			},
			expected: output{
				changed: `
ARG RELEASEID=1809
# NB(thxCode): automatic platform ARGs, ref to:
# - https://docs.docker.com/engine/reference/builder/#automatic-platform-args-in-the-global-scope
ARG TARGETPLATFORM
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT

FROM mcr.microsoft.com/windows/servercore:${RELEASEID} as builder

ENTRYPOINT ["powershell.exe", "-NoLogo"]
`,
			},
		},
	}

	for _, tc := range testCases {
		var actual = InjectTargetPlatformArgsToDockerfile(tc.given.raw, tc.given.os, tc.given.arch, tc.given.variant)
		var actualString string
		if buf, ok := actual.(*bytes.Buffer); ok {
			actualString = buf.String()
		} else {
			var bf bytes.Buffer
			_, _ = bf.ReadFrom(actual)
			actualString = bf.String()
		}
		assert.Equal(t, tc.expected.changed, actualString, "case %q", tc.name)
	}
}
