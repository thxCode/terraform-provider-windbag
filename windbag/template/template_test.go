package template

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/stretchr/testify/assert"
)

func TestTryRender(t *testing.T) {
	// NB(thxCode): respect the Terraform Acceptance logic.
	if os.Getenv(resource.TestEnvVar) != "" {
		t.Skip(fmt.Sprintf(
			"Unit tests skipped as env '%s' set",
			resource.TestEnvVar))
		return
	}

	type tmplData struct {
		Version                       string
		DownloadURI                   string
		AllowNonDistributableArtifact []string
		Experimental                  bool
		MaxConcurrentDownloads        int
		MaxConcurrentUploads          int
		MaxDownloadAttempts           int
		RegistryMirrors               []string
	}
	var defaultTmpl = `
{{- if .Version }}
$env:DOCKER_VERSION="{{ .Version }}";
{{- end }}
{{- if .DownloadURI }}
$env:DOCKER_DOWNLOAD_URI="{{ .DownloadURI }}";
{{- end }}
{{- if .AllowNonDistributableArtifact }}
$env:DOCKER_CONFIGURATION_ALLOW_NONDISTRIBUTABLE_ARTIFACT="{{ .AllowNonDistributableArtifact | join "," }}";
{{- end }}
$env:DOCKER_CONFIGURATION_EXPERIMENTAL="{{ .Experimental | toString }}";
{{- if .MaxConcurrentDownloads }}
$env:DOCKER_CONFIGURATION_MAX_CONCURRENT_DOWNLOADS="{{ .MaxConcurrentDownloads }}";
{{- end }}
{{- if .MaxConcurrentUploads }}
$env:DOCKER_CONFIGURATION_MAX_CONCURRENT_UPLOADS="{{ .MaxConcurrentUploads }}";
{{- end }}
{{- if .MaxDownloadAttempts }}
$env:DOCKER_CONFIGURATION_MAX_DOWNLOAD_ATTEMPTS="{{ .MaxDownloadAttempts }}";
{{- end }}
{{- if .RegistryMirrors }}
$env:DOCKER_CONFIGURATION_REGISTRY_MIRRORS="{{ .RegistryMirrors | join "," }}";
{{- end }}
Invoke-WebRequest -UseBasicParsing -Uri https://raw.githubusercontent.com/thxCode/terraform-provider-windbag/master/tools/docker.ps1 | Invoke-Expression;
`

	type input struct {
		data interface{}
		tmpl string
	}
	type output struct {
		render string
	}

	var testCases = []struct {
		name     string
		given    input
		expected output
	}{
		{
			name: "previous default version",
			given: input{
				data: &tmplData{
					Version: "19.03",
				},
				tmpl: defaultTmpl,
			},
			expected: output{
				render: `
$env:DOCKER_VERSION="19.03";
$env:DOCKER_CONFIGURATION_EXPERIMENTAL="false";
Invoke-WebRequest -UseBasicParsing -Uri https://raw.githubusercontent.com/thxCode/terraform-provider-windbag/master/tools/docker.ps1 | Invoke-Expression;
`,
			},
		},
		{
			name: "new default version",
			given: input{
				data: &tmplData{
					Version:                "19.03",
					Experimental:           true,
					MaxConcurrentDownloads: 8,
					MaxConcurrentUploads:   8,
					MaxDownloadAttempts:    10,
				},
				tmpl: defaultTmpl,
			},
			expected: output{
				render: `
$env:DOCKER_VERSION="19.03";
$env:DOCKER_CONFIGURATION_EXPERIMENTAL="true";
$env:DOCKER_CONFIGURATION_MAX_CONCURRENT_DOWNLOADS="8";
$env:DOCKER_CONFIGURATION_MAX_CONCURRENT_UPLOADS="8";
$env:DOCKER_CONFIGURATION_MAX_DOWNLOAD_ATTEMPTS="10";
Invoke-WebRequest -UseBasicParsing -Uri https://raw.githubusercontent.com/thxCode/terraform-provider-windbag/master/tools/docker.ps1 | Invoke-Expression;
`,
			},
		},
		{
			name: "new advance version",
			given: input{
				data: &tmplData{
					Version: "19.03",
					AllowNonDistributableArtifact: []string{
						"registry.cn-hangzhou.aliyuncs.com",
						"registry.cn-hongkong.aliyuncs.com",
					},
					Experimental:           true,
					MaxConcurrentDownloads: 8,
					MaxConcurrentUploads:   8,
					MaxDownloadAttempts:    10,
				},
				tmpl: defaultTmpl,
			},
			expected: output{
				render: `
$env:DOCKER_VERSION="19.03";
$env:DOCKER_CONFIGURATION_ALLOW_NONDISTRIBUTABLE_ARTIFACT="registry.cn-hangzhou.aliyuncs.com,registry.cn-hongkong.aliyuncs.com";
$env:DOCKER_CONFIGURATION_EXPERIMENTAL="true";
$env:DOCKER_CONFIGURATION_MAX_CONCURRENT_DOWNLOADS="8";
$env:DOCKER_CONFIGURATION_MAX_CONCURRENT_UPLOADS="8";
$env:DOCKER_CONFIGURATION_MAX_DOWNLOAD_ATTEMPTS="10";
Invoke-WebRequest -UseBasicParsing -Uri https://raw.githubusercontent.com/thxCode/terraform-provider-windbag/master/tools/docker.ps1 | Invoke-Expression;
`,
			},
		},
	}

	for _, tc := range testCases {
		var actual = TryRender(tc.given.data, tc.given.tmpl)
		assert.Equal(t, tc.expected.render, actual, "case %q", tc.name)
	}
}
