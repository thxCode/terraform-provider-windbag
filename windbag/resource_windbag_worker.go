package windbag

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"

	"github.com/thxcode/terraform-provider-windbag/windbag/template"
	"github.com/thxcode/terraform-provider-windbag/windbag/utils"
	"github.com/thxcode/terraform-provider-windbag/windbag/worker"
	"github.com/thxcode/terraform-provider-windbag/windbag/worker/powershell"
)

func resourceWindbagWorker() *schema.Resource {
	return &schema.Resource{
		Description: "Specify the worker to build.",

		CreateContext: resourceWindbagWorkerCreate,
		ReadContext:   resourceWindbagWorkerRead,
		UpdateContext: resourceWindbagWorkerUpdate,
		DeleteContext: resourceWindbagWorkerDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"address": {
				Description:  "Specify the address of worker, and use the IP part as the resource ID.",
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validationIsIPWithPort,
			},
			"work_dir": {
				Description: "Specify the working directory of worker.",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "C:/etc/windbag",
			},
			"ssh": {
				Description: "Specify to use SSH to login the worker.",
				Type:        schema.TypeSet,
				Optional:    true,
				MinItems:    1,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"username": {
							Description: "Specify the username for authenticating the worker.",
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "root",
							ForceNew:    true,
						},
						"password": {
							Description: "Specify the password for authenticating the worker.",
							Type:        schema.TypeString,
							Optional:    true,
							Sensitive:   true,
							ForceNew:    true,
						},
						"key": {
							Description: "Specify the content of Private Key to authenticate.",
							Type:        schema.TypeString,
							Optional:    true,
							Sensitive:   true,
							ForceNew:    true,
						},
						"cert": {
							Description: "Specify the content of Certificate to authenticate.",
							Type:        schema.TypeString,
							Optional:    true,
							ForceNew:    true,
						},
						"with_agent": {
							Description: "Specify to use ssh-agent to manage the login credential.",
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							ForceNew:    true,
						},
					},
				},
			},
			"os_major": {
				Description: "Observed the major version number of worker.",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"os_minor": {
				Description: "Observed the minor version number of worker.",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"os_build": {
				Description: "Observed the build number of worker.",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"os_ubr": {
				Description: "Observed the UBR version number of worker.",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"os_release": {
				Description: "Observed the release ID of worker.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"os_type": {
				Description: "Observed the type of worker.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"os_arch": {
				Description: "Observed the arch of worker.",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

func resourceWindbagWorkerCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var p = meta.(*provider)
	var id = resourceWindbagWorkerID(utils.ToString(d.Get("address"))) // use the address as the resource ID

	// dail worker
	if _, ok := p.workers[id]; !ok {
		if v, ok := d.GetOk("ssh"); ok {
			var ssh = v.(*schema.Set).List()[0].(map[string]interface{})
			var opts worker.DialSSHOptions

			opts.Address = utils.ToString(d.Get("address"))
			opts.Username = utils.ToString(ssh["username"])
			opts.Password = utils.ToString(ssh["password"])
			if v := utils.ToString(ssh["key"]); v != "" {
				opts.KeyPEMBlockBytes = utils.UnsafeStringToBytes(v)
			}
			if v := utils.ToString(ssh["cert"]); v != "" {
				opts.CertPEMBlockBytes = utils.UnsafeStringToBytes(v)
			}
			opts.WithAgent = utils.ToBool(ssh["with_agent"])

			log.Printf("[DEBUG] Dialing worker %q via SSH\n", id)
			if w, err := worker.DailSSH(opts); err != nil {
				log.Printf("[ERROR] Failed to dial worker %q via SSH: %v\n", id, err)
			} else {
				log.Printf("[DEBUG] Dialed worker %q via SSH\n", id)
				p.workers[id] = w
			}
		}
	}
	var w = p.workers[id]
	if w == nil {
		return diag.Errorf("worker %s is inaccessible", id)
	}

	log.Printf("[DEBUG] Preparing worker %q\n", id)
	var err = w.PowerShell(ctx, nil, func(ctx context.Context, ps *powershell.PowerShell) error {
		var psc, err = ps.Commands()
		if err != nil {
			return errors.Wrap(err, "failed to setup interaction")
		}
		defer func() {
			if err := psc.Close(); err != nil {
				log.Printf("[ERROR] Failed to close interaction: %v\n", err)
			}
		}()

		// prepare host build directory
		var workDir = "C:/etc/windbag"
		if v, ok := d.GetOk("work_dir"); ok {
			workDir = utils.ToString(v)
		}
		var command = template.TryRender(
			map[string]interface{}{
				"WorkDir": workDir,
			},
			`$Path = "{{ .WorkDir }}";
if (Test-Path -Path "$Path/buildpath") {
  if (-not (Test-Path -Path "$Path/buildpath" -PathType Container)) {
    Remove-Item -Force -Path "$Path/buildpath" -ErrorAction Ignore | Out-Null;
  }
};
New-Item -Force -ItemType Directory -Path "$Path/buildpath" | Out-Null;
if (Test-Path -Path "$Path/dockerfile") {
  if (-not (Test-Path -Path "$Path/dockerfile" -PathType Container)) {
    Remove-Item -Force -Path "$Path/dockerfile" -ErrorAction Ignore | Out-Null;
  }
};
New-Item -Force -ItemType Directory -Path "$Path/dockerfile" | Out-Null;
`,
		)
		_, stderr, err := psc.Execute(ctx, id, command)
		if err != nil {
			return errors.Wrap(err, "failed to execute workdir creation")
		}
		if stderr != "" {
			return errors.Errorf("error executing workdir creation: %s", stderr)
		}

		return nil
	})
	if err != nil {
		return diag.Errorf("failed to prepare worker %s: %v", id, err)
	}
	log.Printf("[INFO] Prepared worker %q\n", id)

	d.SetId(id)
	return resourceWindbagWorkerRead(ctx, d, meta)
}

func resourceWindbagWorkerRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var p = meta.(*provider)
	var id = d.Id()

	// dail worker
	if _, ok := p.workers[id]; !ok {
		if v, ok := d.GetOk("ssh"); ok {
			var ssh = v.(*schema.Set).List()[0].(map[string]interface{})
			var opts worker.DialSSHOptions

			opts.Address = utils.ToString(d.Get("address"))
			opts.Username = utils.ToString(ssh["username"])
			opts.Password = utils.ToString(ssh["password"])
			if v := utils.ToString(ssh["key"]); v != "" {
				opts.KeyPEMBlockBytes = utils.UnsafeStringToBytes(v)
			}
			if v := utils.ToString(ssh["cert"]); v != "" {
				opts.CertPEMBlockBytes = utils.UnsafeStringToBytes(v)
			}
			opts.WithAgent = utils.ToBool(ssh["with_agent"])

			log.Printf("[DEBUG] Dialing worker %q via SSH\n", id)
			if w, err := worker.DailSSH(opts); err != nil {
				log.Printf("[ERROR] Failed to dial worker %q via SSH: %v\n", id, err)
			} else {
				log.Printf("[DEBUG] Dialed worker %q via SSH\n", id)
				p.workers[id] = w
			}
		}
	}
	var w = p.workers[id]
	if w == nil {
		return diag.Errorf("worker %s is inaccessible", id)
	}

	// retrieve information
	log.Printf("[DEBUG] Retrieving worker %q\n", id)
	var os = make(map[string]interface{})
	var err = w.PowerShell(ctx, nil, func(ctx context.Context, ps *powershell.PowerShell) error {
		var psc, err = ps.Commands()
		if err != nil {
			return errors.Wrap(err, "failed to setup interaction")
		}
		defer func() {
			if err := psc.Close(); err != nil {
				log.Printf("[ERROR] Failed to close interaction: %v\n", err)
			}
		}()

		// get host version information
		var command = `Get-ItemProperty -Path "HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion" | Select-Object -Property CurrentMajorVersionNumber,CurrentMinorVersionNumber,CurrentBuildNumber,UBR,ReleaseId,BuildLabEx,CurrentBuild | ConvertTo-JSON -Compress;`
		stdout, stderr, err := psc.Execute(ctx, id, command)
		if err != nil {
			return errors.Wrap(err, "failed to retrieve host version")
		}
		if stderr != "" {
			return errors.Errorf("error retrieving host version: %s", stderr)
		}
		var hostVersion map[string]interface{}
		if err := utils.UnmarshalJSON(utils.UnsafeStringToBytes(stdout), &hostVersion); err != nil {
			return errors.Wrap(err, "failed to unmarshal host version retrieve output")
		}
		os["os_major"] = utils.ToInt(hostVersion["CurrentMajorVersionNumber"])
		os["os_minor"] = utils.ToInt(hostVersion["CurrentMinorVersionNumber"])
		os["os_build"] = utils.ToInt(hostVersion["CurrentBuildNumber"])
		os["os_ubr"] = utils.ToInt(hostVersion["UBR"])
		os["os_release"] = utils.ToString(hostVersion["ReleaseId"])
		os["os_type"] = "windows"

		// get host arch information
		command = `[Environment]::GetEnvironmentVariable("PROCESSOR_ARCHITECTURE", [EnvironmentVariableTarget]::Machine);`
		stdout, stderr, err = psc.Execute(ctx, id, command)
		if err != nil {
			return errors.Wrap(err, "failed to retrieve host arch")
		}
		if stderr != "" {
			return errors.Errorf("error retrieving host arch: %s", stderr)
		}
		var hostArch = strings.ToLower(strings.TrimSpace(stdout))
		switch hostArch {
		case "arm":
			os["os_arch"] = "arm"
		case "x86", "386":
			os["os_arch"] = "386"
		default:
			os["os_arch"] = "amd64"
		}

		return nil
	})
	if err != nil {
		return diag.Errorf("failed to retrieve worker %s: %v", id, err)
	}
	for osKey, osValue := range os {
		if err = d.Set(osKey, osValue); err != nil {
			return diag.Errorf("failed to set computed '%s' attribute of worker %s: %v", osKey, id, err)
		}
	}
	log.Printf("[INFO] Retrieved worker %q\n", id)

	return nil
}

func resourceWindbagWorkerUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var p = meta.(*provider)
	var id = d.Id()
	if w, ok := p.workers[id]; !ok {
		_ = w.Close()
		delete(p.workers, id)
	}
	d.SetId("")
	return resourceWindbagImageCreate(ctx, d, meta) // recreate
}

func resourceWindbagWorkerDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var p = meta.(*provider)
	var id = d.Id()
	if w, ok := p.workers[id]; ok {
		var err = w.Close()
		log.Printf("[WARN] Failed to close deleted worker %q: %v\n", id, err)
		delete(p.workers, id)
	}
	d.SetId("")
	return nil
}

func resourceWindbagWorkerID(addr string) string {
	if idx := strings.LastIndex(addr, ":"); idx > 0 {
		return addr[:idx]
	}
	return addr
}

func validationIsIPWithPort(i interface{}, k string) (warnings []string, errors []error) {
	var v, ok = i.(string)
	if !ok {
		errors = append(errors, fmt.Errorf("expected type of %s to be string", k))
		return warnings, errors
	}

	if idx := strings.Index(v, ":"); idx < 0 {
		errors = append(errors, fmt.Errorf("expected %s to be a URL in form of ip:port", k))
		return warnings, errors
	} else if v[idx+1:] != "22" {
		warnings = append(warnings, fmt.Sprintf("the default port of SSH protocol is 22, but got %s in %s", v[idx+1:], k))
	}

	return warnings, errors
}
