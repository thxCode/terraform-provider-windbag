package windbag

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"github.com/thxcode/terraform-provider-windbag/windbag/dial"
	"github.com/thxcode/terraform-provider-windbag/windbag/dial/powershell"
	"github.com/thxcode/terraform-provider-windbag/windbag/docker"
	"github.com/thxcode/terraform-provider-windbag/windbag/log"
	"github.com/thxcode/terraform-provider-windbag/windbag/template"
	"github.com/thxcode/terraform-provider-windbag/windbag/utils"
)

func resourceWindbagImage() *schema.Resource {
	return &schema.Resource{
		Description: "Specify the image to build.",

		CreateContext: resourceWindbagImageCreate,
		ReadContext:   resourceWindbagImageRead,
		UpdateContext: resourceWindbagImageUpdate,
		DeleteContext: resourceWindbagImageDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Timeouts: &schema.ResourceTimeout{
			Default: schema.DefaultTimeout(30 * time.Minute),
			Create:  schema.DefaultTimeout(1 * time.Hour),
			Read:    schema.DefaultTimeout(2 * time.Hour),
			Update:  schema.DefaultTimeout(2 * time.Hour),
			Delete:  schema.DefaultTimeout(1 * time.Hour),
		},

		Schema: map[string]*schema.Schema{
			"build_arg": {
				Description: "Specify the build-time arguments.",
				Type:        schema.TypeMap,
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"build_arg_release_mapper": {
				Description: "Specify the release related build-time arguments mapper.",
				Type:        schema.TypeSet,
				Optional:    true,
				Set: func(i interface{}) int {
					var m = utils.ToStringInterfaceMap(i)
					var release = utils.ToString(m["release"])
					return utils.HashString(release)
				},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"release": {
							Description: "Specify the release ID of worker.",
							Type:        schema.TypeString,
							Required:    true,
						},
						"build_arg": {
							Description: "Specify the build-time arguments of related release.",
							Type:        schema.TypeMap,
							Optional:    true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},
			"disable_target_platform_args_injection": {
				Description: "Specify whether to disable the target platform arguments injection, ref to https://registry.terraform.io/providers/thxCode/windbag/latest/docs#highlight.",
				Type:        schema.TypeBool,
				Optional:    true,
			},
			"disable_release_build_args_injection": {
				Description: "Specify whether to disable the release related build arguments injection, ref to https://registry.terraform.io/providers/thxCode/windbag/latest/docs#highlight.",
				Type:        schema.TypeBool,
				Optional:    true,
			},
			"file": {
				Description: "Specify the path of the building Dockerfile.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"force_rm": {
				Description: "Specify to remove intermediate containers.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
			"isolation": {
				Description:  "Specify the isolation technology of container.",
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"default", "hyperv", "process"}, false),
			},
			"label": {
				Description: "Specify the metadata label.",
				Type:        schema.TypeMap,
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"no_cache": {
				Description: "Specify the isolation technology of container.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
			"rm": {
				Description: "Specify to remove intermediate containers after a successful build.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
			},
			"tag": {
				Description: "Specify the name of the built artifact, and use the repository of the last item as this resource ID.",
				Type:        schema.TypeList,
				Required:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"target": {
				Description: "Specify the target of build stage to build.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"path": {
				Description: "Specify the path to build.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"push": {
				Description: "Specify to push the build artifact.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
			},
			"push_timeout": {
				Description: "Specify the timeout to push pre build tag.",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "15m",
			},
			"manifest": {
				Description: "Specify to manifest the build artifact.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
			},
			"manifest_timeout": {
				Description: "Specify the timeout to manifest pre tag.",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "15m",
			},
			"registry": {
				Description: "Specify the authentication registry of registry.",
				Type:        schema.TypeSet,
				Optional:    true,
				Set: func(i interface{}) int {
					var m = utils.ToStringInterfaceMap(i)
					var release = utils.ToString(m["address"])
					return utils.HashString(release)
				},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"address": {
							Description: "Specify the address of the registry.",
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "docker.io",
							ForceNew:    true,
						},
						"username": {
							Description: "Specify the username of the registry credential.",
							Type:        schema.TypeString,
							Required:    true,
							ForceNew:    true,
						},
						"password": {
							Description: "Specify the password of the registry credential.",
							Type:        schema.TypeString,
							Required:    true,
							ForceNew:    true,
							Sensitive:   true,
						},
						"login_timeout": {
							Description: "Specify the timeout to login.",
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "5m",
							ForceNew:    true,
						},
					},
				},
			},
			"worker": {
				Description: "Specify the workers to build.",
				Type:        schema.TypeSet,
				Required:    true,
				Set: func(i interface{}) int {
					var m = utils.ToStringInterfaceMap(i)
					var release = utils.ToString(m["address"])
					return utils.HashString(release)
				},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"address": {
							Description:  "Specify the address of worker.",
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validationWindbagImageWorkerAddress,
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
							Required:    true,
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
									"retry_timeout": {
										Description: "Specify the timeout to retry dialing.",
										Type:        schema.TypeString,
										Optional:    true,
										Default:     "10m",
										ForceNew:    true,
									},
								},
							},
						},
						"build_context": {
							Description: "Observed the build context of worker.",
							Type:        schema.TypeSet,
							Computed:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"dockerfile": {
										Description: "Observed the dockerfile of build context",
										Type:        schema.TypeString,
										Computed:    true,
									},
									"buildpath": {
										Description: "Observed the buildpath of build context",
										Type:        schema.TypeString,
										Computed:    true,
									},
								},
							},
						},
						"build_information": {
							Description: "Observed the build information of worker.",
							Type:        schema.TypeSet,
							Computed:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
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
									"os_arch": {
										Description: "Observed the arch of worker.",
										Type:        schema.TypeString,
										Computed:    true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func resourceWindbagImageCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var id = func() string {
		var tags = utils.ToStringSlice(d.Get("tag"))
		return resourceWindbagImageID(tags[len(tags)-1]) // use the last item as the resource ID
	}()

	log.Infof("==== %s dialing all workers ====", id)
	var p = meta.(*provider)
	var workers = utils.ToInterfaceSlice(d.Get("worker"))
	var workerDialers = make(map[string]dial.Dialer, len(workers))
	// allow pushing foreign layers
	if p.docker.AllowNonDistributableArtifact != nil {
		var regAddresses []string
		for _, r := range utils.ToInterfaceSlice(d.Get("registry")) {
			var reg = utils.ToStringInterfaceMap(r)
			var regAddress = utils.ToString(reg["address"])
			regAddresses = append(regAddresses, regAddress)
		}
		p.docker.AllowNonDistributableArtifact = regAddresses
	}
	for _, w := range workers {
		var worker = utils.ToStringInterfaceMap(w)
		var workerAddress = utils.ToString(worker["address"])
		var workerSSH = utils.ToStringInterfaceMap(worker["ssh"])
		var workerDialer, err = p.dialWorkerBySSH(ctx, id, workerAddress, workerSSH, true)
		if err != nil {
			return diag.Errorf("failed to dial worker %s via SSH: %v", workerAddress, err)
		}
		workerDialers[workerAddress] = workerDialer
	}
	defer func() {
		for _, workerDial := range workerDialers {
			_ = workerDial.Close()
		}
	}()
	log.Infof("==== %s dialed all workers ====", id)

	var buildpath, err = func(p string) (dPath string, dErr error) {
		dPath, dErr = utils.NormalizePath(p)
		if dErr != nil {
			dErr = errors.Wrapf(dErr, "path %q could not be normalized", p)
		} else {
			if stat, err := os.Stat(dPath); err != nil {
				dErr = errors.Errorf("path %q is not existed", dPath)
			} else if !stat.IsDir() {
				dErr = errors.Errorf("path %q is not a directory", dPath)
			}
		}
		return dPath, dErr
	}(utils.ToString(d.Get("path")))
	if err != nil {
		return diag.Errorf("failed to get the buildpath of image %s: %v", id, err)
	}

	dockerfilePath, err := func(p string) (fPath string, fErr error) {
		if p == "" {
			p = filepath.Join(buildpath, "Dockerfile")
		}
		fPath, fErr = utils.NormalizePath(p)
		if fErr != nil {
			fErr = errors.Wrapf(fErr, "path %q could not be normalized", p)
		} else {
			if stat, err := os.Stat(fPath); err != nil {
				fErr = errors.Errorf("path %q is not existed", fPath)
			} else if stat.IsDir() {
				fErr = errors.Errorf("path %q is not a file", fPath)
			}
		}
		return fPath, fErr
	}(utils.ToString(d.Get("file")))
	if err != nil {
		return diag.Errorf("failed to get the dockerfile of image %s: %v", id, err)
	}

	/*
		construct context and retrieve information
	*/

	log.Infof("==== %s shipping build context to all workers ====", id)
	for _, w := range workers {
		var buildWorker = utils.ToStringInterfaceMap(w)
		var workerAddress = utils.ToString(buildWorker["address"])
		var workerID = fmt.Sprintf("%s/%s", workerAddress, id)
		var workerWorkDir = utils.ToString(buildWorker["work_dir"])

		// retrieve information
		if info := utils.ToStringInterfaceMap(buildWorker["build_information"]); len(info) == 0 {
			var workerDialer = workerDialers[workerAddress]
			var err = workerDialer.PowerShell(ctx, nil, func(ctx context.Context, ps *powershell.PowerShell) error {
				var psc, err = ps.Commands()
				if err != nil {
					return errors.Wrap(err, "failed to setup interaction")
				}
				defer func() {
					if err := psc.Close(); err != nil {
						log.Errorf("Failed to close interaction: %v", err)
					}
				}()

				// get host release
				var command = `Get-ItemProperty -Path "HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion" | Select-Object -Property CurrentMajorVersionNumber,CurrentMinorVersionNumber,CurrentBuildNumber,UBR,ReleaseId,BuildLabEx,CurrentBuild | ConvertTo-JSON -Compress;`
				stdout, stderr, err := psc.Execute(ctx, workerID, command)
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
				info["os_major"] = utils.ToInt(hostVersion["CurrentMajorVersionNumber"])
				info["os_minor"] = utils.ToInt(hostVersion["CurrentMinorVersionNumber"])
				info["os_build"] = utils.ToInt(hostVersion["CurrentBuildNumber"])
				info["os_ubr"] = utils.ToInt(hostVersion["UBR"])
				info["os_release"] = utils.ToString(hostVersion["ReleaseId"])

				// get host arch
				command = `[Environment]::GetEnvironmentVariable("PROCESSOR_ARCHITECTURE", [EnvironmentVariableTarget]::Machine);`
				stdout, stderr, err = psc.Execute(ctx, workerID, command)
				if err != nil {
					return errors.Wrap(err, "failed to retrieve host arch")
				}
				if stderr != "" {
					return errors.Errorf("error retrieving host arch: %s", stderr)
				}
				info["os_arch"] = func() string {
					var hostArch = strings.ToLower(strings.TrimSpace(stdout))
					switch hostArch {
					case "arm":
						return "arm"
					case "x86", "386":
						return "386"
					default:
						return "amd64"
					}
				}()

				return nil
			})
			if err != nil {
				return diag.Errorf("failed to retrieve information on worker %s: %v", workerAddress, err)
			}
			buildWorker["build_information"].(*schema.Set).Add(info)
		}

		// construct context
		if info := utils.ToStringInterfaceMap(buildWorker["build_context"]); len(info) == 0 {
			var workerDialer = workerDialers[workerAddress]
			var err = workerDialer.PowerShell(ctx, nil, func(ctx context.Context, ps *powershell.PowerShell) error {
				var psc, err = ps.Commands()
				if err != nil {
					return errors.Wrap(err, "failed to setup interaction")
				}
				defer func() {
					if err := psc.Close(); err != nil {
						log.Errorf("Failed to close interaction: %v", err)
					}
				}()

				// prepare host build directory
				command := template.TryRender(
					map[string]interface{}{
						"WorkDir": workerWorkDir,
					},
					`
$Path = "{{ .WorkDir }}";
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
				_, stderr, err := psc.Execute(ctx, workerID, command)
				if err != nil {
					return errors.Wrap(err, "failed to execute workdir creation")
				}
				if stderr != "" {
					return errors.Errorf("error executing workdir creation: %s", stderr)
				}

				// transfer build path archive
				buildpathArchive, err := docker.GetBuildpathArchive(buildpath, dockerfilePath)
				if err != nil {
					return errors.Wrap(err, "failed to retrieve the buildpath")
				}
				var buildpathArchiveShippedDst = filepath.Join(workerWorkDir, "buildpath", fmt.Sprintf("%s.zip", id))
				_, err = workerDialer.Copy(ctx, buildpathArchive, buildpathArchiveShippedDst)
				if err != nil {
					return errors.Wrapf(err, "failed to ship the buildpath to worker %s", workerAddress)
				}
				// expand build path archive
				var buildpathArchiveExpandDst = filepath.Join(workerWorkDir, "buildpath", id)
				command = template.TryRender(
					map[string]interface{}{
						"Src": buildpathArchiveShippedDst,
						"Dst": buildpathArchiveExpandDst,
					},
					`Expand-Archive -Force -Path "{{ .Src }}" -DestinationPath "{{ .Dst }}" | Out-Null`,
				)
				_, stderr, err = psc.Execute(ctx, workerID, command)
				if err != nil {
					return errors.Wrap(err, "failed to execute docker buildpath archive expansion")
				}
				if stderr != "" {
					return errors.Errorf("error executing docker buildpath archive expansion: %s", stderr)
				}
				info["buildpath"] = buildpathArchiveExpandDst

				// transfer build dockerfile
				var dockerfile io.Reader
				if !utils.ToBool(d.Get("disable_target_platform_args_injection")) {
					var f, _ = os.Open(dockerfilePath)
					var buildInfo = utils.ToStringInterfaceMap(buildWorker["build_information"])
					var (
						targetType    = "windows"
						targetArch    = utils.ToString(buildInfo["os_arch"])
						targetVariant = utils.ToString(buildInfo["os_release"])
					)
					dockerfile = docker.InjectTargetPlatformArgsToDockerfile(f, targetType, targetArch, targetVariant)
					_ = f.Close()
				} else {
					var f, _ = os.Open(dockerfilePath)
					defer func() { _ = f.Close() }()
					dockerfile = f
				}
				var dockerfileShippedDst = filepath.Join(workerWorkDir, "dockerfile", fmt.Sprintf("Dockerfile.%s", id))
				_, err = workerDialer.Copy(ctx, dockerfile, dockerfileShippedDst)
				if err != nil {
					return errors.Wrapf(err, "failed to ship the dockerfile to worker %s", workerAddress)
				}
				info["dockerfile"] = dockerfileShippedDst

				return nil
			})
			if err != nil {
				return diag.Errorf("failed to create build context on worker %s: %v", workerAddress, err)
			}
			buildWorker["build_context"].(*schema.Set).Add(info)
		}
	}
	log.Infof("==== %s shipped build context to all workers ====", id)

	/*
		login registries
	*/

	log.Infof("==== %s logging all registries on all workers ====", id)
	var registryLoginCommands = make(map[string]string)
	for _, r := range utils.ToInterfaceSlice(d.Get("registry")) {
		var reg = utils.ToStringInterfaceMap(r)
		var regAddress = utils.ToString(reg["address"])
		var regUsername = utils.ToString(reg["username"])
		var regPassword = utils.ToString(reg["password"])

		var command = docker.ConstructRegistryLoginCommand(regAddress, regUsername, regPassword)
		registryLoginCommands[regAddress] = command
	}
	if len(registryLoginCommands) != 0 {
		var eg, egctx = errgroup.WithContext(ctx)
		for _, w := range workers {
			var loginWorker = utils.ToStringInterfaceMap(w)
			var workerAddress = utils.ToString(loginWorker["address"])
			var workerLoginTimeout = utils.ToDuration(loginWorker["login_timeout"], 5*time.Minute)
			var workerID = fmt.Sprintf("%s/%s", workerAddress, id)

			// docker login
			eg.Go(func() error {
				var workerDialer = workerDialers[workerAddress]
				err = workerDialer.PowerShell(egctx, nil, func(ctx context.Context, ps *powershell.PowerShell) error {
					var psc, err = ps.Commands()
					if err != nil {
						return errors.Wrap(err, "failed to setup interaction")
					}
					defer func() {
						if err := psc.Close(); err != nil {
							log.Errorf("Failed to close interaction: %v", err)
						}
					}()

					for reg := range registryLoginCommands {
						var err = resource.RetryContext(egctx, workerLoginTimeout, func() *resource.RetryError {
							var command = registryLoginCommands[reg]
							var stdout, stderr, err = psc.Execute(ctx, workerID, command)
							if err != nil {
								log.Errorf("Failed to login registry %q on worker %q", reg, workerAddress)
								return resource.RetryableError(errors.Wrapf(err, "failed to log registry %s", reg))
							}
							if stderr != "" {
								if !strings.HasPrefix(stdout, "Login Succeeded") {
									log.Errorf("Failed to login registry %q on worker %q", reg, workerAddress)
									return resource.RetryableError(errors.Errorf("failed to login registry %s: %v", reg, stderr))
								}
							}
							return nil
						})
						if err != nil {
							return err
						}
						log.Infof("Logon registry %q on worker %q\n", reg, workerAddress)
					}
					return nil
				})
				if err != nil {
					return errors.Wrapf(err, "error executing docker-login command on worker %s", workerAddress)
				}
				return nil
			})

		}
		if err := eg.Wait(); err != nil {
			return diag.Errorf("failed to login registry for image %s: %v", id, err)
		}
	}
	log.Infof("==== %s logon all registries on all workers ====", id)

	d.SetId(id)
	return resourceWindbagImageRead(ctx, d, meta)
}

func resourceWindbagImageRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var p = meta.(*provider)
	var id = d.Id()

	log.Infof("==== %s dialing all workers ====", id)
	var workers = utils.ToInterfaceSlice(d.Get("worker"))
	var workerDialers = make(map[string]dial.Dialer, len(workers))
	for _, w := range workers {
		var worker = utils.ToStringInterfaceMap(w)
		var workerAddress = utils.ToString(worker["address"])
		var workerSSH = utils.ToStringInterfaceMap(worker["ssh"])
		var workerDialer, err = p.dialWorkerBySSH(ctx, id, workerAddress, workerSSH, false)
		if err != nil {
			return diag.Errorf("failed to dial worker %s via SSH: %v", workerAddress, err)
		}
		workerDialers[workerAddress] = workerDialer
	}
	defer func() {
		for _, workerDial := range workerDialers {
			_ = workerDial.Close()
		}
	}()
	log.Infof("==== %s dialed all workers ====", id)

	/*
		build
	*/

	log.Infof("==== %s building on all workers ====", id)
	var buildOpts = types.ImageBuildOptions{
		Version:     types.BuilderV1,
		Tags:        utils.ToStringSlice(d.Get("tag")),
		Labels:      utils.ToStringStringMap(d.Get("label")),
		ForceRemove: utils.ToBool(d.Get("force_rm")),
		Isolation:   container.Isolation(utils.ToString(d.Get("isolation"))),
		NoCache:     utils.ToBool(d.Get("no_cache")),
		Remove:      utils.ToBool(d.Get("rm")),
		Target:      utils.ToString(d.Get("target")),
		BuildArgs: func() map[string]*string {
			var args = map[string]*string{}
			for argName, argVal := range utils.ToStringStringMap(d.Get("build_arg")) {
				args[argName] = &argVal
			}
			return args
		}(),
	}
	var extraBuildArgsMapper = make(map[string]map[string]*string)
	for _, mapper := range utils.ToInterfaceSlice(d.Get("release_build_arg_mapper")) {
		var buildArgsMapper = utils.ToStringInterfaceMap(mapper)
		var buildRelease = utils.ToString(buildArgsMapper["release"])
		var buildArgs = make(map[string]*string)
		for k, v := range utils.ToStringStringMap(buildArgsMapper["build_arg"]) {
			buildArgs[k] = &v
		}
		extraBuildArgsMapper[buildRelease] = buildArgs
	}
	eg, egctx := errgroup.WithContext(ctx)
	for _, w := range workers {
		var buildWorker = utils.ToStringInterfaceMap(w)
		var workerAddress = utils.ToString(buildWorker["address"])
		var workerID = fmt.Sprintf("%s/%s", workerAddress, id)

		// docker build
		eg.Go(func() error {
			log.Infof("Building image %q on worker %q", id, workerAddress)
			var workerDialer = workerDialers[workerAddress]
			var err = workerDialer.PowerShell(egctx, nil, func(ctx context.Context, ps *powershell.PowerShell) error {
				var psc, err = ps.Commands()
				if err != nil {
					return errors.Wrap(err, "failed to setup interaction")
				}
				defer func() {
					if err := psc.Close(); err != nil {
						log.Errorf("Failed to close interaction: %v", err)
					}
				}()

				var workerOS = "windows"
				var workerBuildInformation = utils.ToStringInterfaceMap(buildWorker["build_information"])
				var workerRelease = utils.ToString(workerBuildInformation["os_release"])
				var workerArch = utils.ToString(workerBuildInformation["os_arch"])
				var workerTagSuffix = fmt.Sprintf("%s-%s-%s", workerOS, workerArch, workerRelease)
				var workerBuildContext = utils.ToStringInterfaceMap(buildWorker["build_context"])

				var command = func(opts types.ImageBuildOptions) string {
					// append build-args
					var buildArgs = make(map[string]*string, len(opts.BuildArgs))
					for k, v := range buildArgs {
						buildArgs[k] = v
					}
					// NB(thxCode): Deprecated, replace with WINDBAGRELEASE
					buildArgs["RELEASEID"] = &workerRelease
					buildArgs["WINDBAGRELEASE"] = &workerRelease
					if extraBuildArgs, exist := extraBuildArgsMapper[workerRelease]; exist {
						for k, v := range extraBuildArgs {
							buildArgs["WINDBAGRELEASE_"+k] = v
						}
					}
					opts.BuildArgs = buildArgs
					// redirect dockerfile
					opts.Dockerfile = utils.ToString(workerBuildContext["dockerfile"])
					// redirect tag
					var tags = make([]string, 0, len(opts.Tags))
					for ti := range opts.Tags {
						tags = append(tags, fmt.Sprintf("%s-%s", opts.Tags[ti], workerTagSuffix))
					}
					opts.Tags = tags
					// render
					return docker.ConstructBuildCommand(opts, utils.ToString(workerBuildContext["buildpath"]))
				}(buildOpts)
				_, stderr, err := psc.Execute(ctx, workerID, command)
				if err != nil {
					return errors.Wrap(err, "failed to execute docker building")
				}
				if stderr != "" {
					return errors.Errorf("error executing docker building: %s", stderr)
				}

				return nil
			})
			if err != nil {
				return errors.Wrapf(err, "error executing docker-build command on worker %s", workerAddress)
			}
			log.Infof("Built image %q on worker %q", id, workerAddress)
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return diag.Errorf("failed to build image %s: %v", id, err)
	}
	log.Infof("==== %s built on all workers ====", id)

	/*
		push
	*/

	if !utils.ToBool(d.Get("push")) {
		log.Warnf(" Skipped to push the image %q", id)
		return nil
	}
	log.Infof("==== %s pushing on all workers ====", id)
	var workerPushTimeout = utils.ToDuration(d.Get("push_timeout"), 15*time.Minute)
	eg, egctx = errgroup.WithContext(ctx)
	for _, w := range workers {
		var pushWorker = utils.ToStringInterfaceMap(w)
		var workerAddress = utils.ToString(pushWorker["address"])
		var workerID = fmt.Sprintf("%s/%s", workerAddress, id)

		// docker push
		eg.Go(func() error {
			var workerDialer = workerDialers[workerAddress]
			var err = workerDialer.PowerShell(egctx, nil, func(ctx context.Context, ps *powershell.PowerShell) error {
				var psc, err = ps.Commands()
				if err != nil {
					return errors.Wrap(err, "failed to setup interaction")
				}
				defer func() {
					if err := psc.Close(); err != nil {
						log.Errorf("Failed to close interaction: %v", err)
					}
				}()

				var workerBuildInformation = utils.ToStringInterfaceMap(pushWorker["build_information"])
				var workerRelease = utils.ToString(workerBuildInformation["os_release"])
				var workerArch = utils.ToString(workerBuildInformation["os_arch"])
				var workerTagSuffix = fmt.Sprintf("windows-%s-%s", workerArch, workerRelease)

				// push tags one by one
				for ti := range buildOpts.Tags {
					var tag = buildOpts.Tags[ti]
					tag = fmt.Sprintf("%s-%s", tag, workerTagSuffix)
					err = resource.RetryContext(egctx, workerPushTimeout, func() *resource.RetryError {
						var command = docker.ConstructImagePushCommand(tag)
						_, stderr, err := psc.Execute(ctx, workerID, command)
						if err != nil {
							log.Errorf("Failed to push image %q on worker %s: %v", tag, workerAddress, err)
							return resource.RetryableError(errors.Wrapf(err, "failed to push image %s", tag))
						}
						if stderr != "" {
							log.Errorf("Failed to push image %q on worker %s: %v", tag, workerAddress, stderr)
							return resource.RetryableError(errors.Errorf("failed to push image %s: %v", tag, stderr))
						}
						return nil
					})
					if err != nil {
						return err
					}
				}
				return nil
			})
			if err != nil {
				return errors.Wrapf(err, "error executing docker-push command of image %s on worker %s", id, workerAddress)
			}
			log.Infof("Pushed image %q on worker %q", id, workerAddress)
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return diag.Errorf("failed to push image %s: %v", id, err)
	}
	log.Infof("==== %s pushed on all workers ====", id)

	/*
		manifest
	*/

	if !utils.ToBool(d.Get("manifest")) {
		log.Warnf(" Skipped to push the image %q", id)
		return nil
	}
	log.Infof("==== %s manifesting on the highest worker ====", id)
	var workerManifestTimeout = utils.ToDuration(d.Get("manifest_timeout"), 15*time.Minute)
	var manifestWorker, tagSuffixes = func() (manifestWorker map[string]interface{}, tagSuffixes []string) {
		var manifestWorkerBuild int
		for _, w := range workers {
			var checkpoint = utils.ToStringInterfaceMap(w)
			var checkpointBuildInformation = utils.ToStringInterfaceMap(checkpoint["build_information"])
			var checkpointOSBuild = utils.ToInt(checkpointBuildInformation["os_build"])
			var checkpointOSArch = utils.ToString(checkpointBuildInformation["os_arch"])
			var checkpointOSRelease = utils.ToString(checkpointBuildInformation["os_release"])

			tagSuffixes = append(tagSuffixes, fmt.Sprintf("windows-%s-%s", checkpointOSArch, checkpointOSRelease))
			if manifestWorker == nil {
				manifestWorker = checkpoint
				manifestWorkerBuild = checkpointOSBuild
			} else if manifestWorkerBuild < checkpointOSBuild {
				manifestWorker = checkpoint
				manifestWorkerBuild = checkpointOSBuild
			}
		}
		return manifestWorker, tagSuffixes
	}()
	var workerAddress = utils.ToString(manifestWorker["address"])
	var workerID = fmt.Sprintf("%s/%s", workerAddress, id)
	eg, egctx = errgroup.WithContext(ctx)
	for ti := range buildOpts.Tags {
		var tag = buildOpts.Tags[ti]
		var manifests []string
		for tsi := range tagSuffixes {
			manifests = append(manifests, fmt.Sprintf("%s-%s", tag, tagSuffixes[tsi]))
		}

		// docker manifest
		eg.Go(func() error {
			log.Infof("Manifesting image %q on worker %q", id, workerAddress)
			var workerDialer = workerDialers[workerAddress]
			var err = resource.RetryContext(egctx, workerManifestTimeout, func() *resource.RetryError {
				var err = workerDialer.PowerShell(egctx, nil, func(ctx context.Context, ps *powershell.PowerShell) error {
					var psc, err = ps.Commands()
					if err != nil {
						return errors.Wrap(err, "failed to setup interaction")
					}
					defer func() {
						if err := psc.Close(); err != nil {
							log.Errorf("Failed to close interaction: %v", err)
						}
					}()

					// manifest create
					var command = docker.ConstructManifestCreateCommand(tag, manifests...)
					_, stderr, err := psc.Execute(ctx, workerID, command)
					if err != nil {
						return errors.Wrap(err, "failed to execute docker manifest creation")
					}
					if stderr != "" {
						return errors.Errorf("error executing docker manifest creation: %s", stderr)
					}

					// manifest push
					command = docker.ConstructManifestPushCommand(tag)
					_, stderr, err = psc.Execute(ctx, workerID, command)
					if err != nil {
						return errors.Wrap(err, "failed to execute docker manifest pushing")
					}
					if stderr != "" {
						return errors.Errorf("error executing docker manifest pushing: %s", stderr)
					}

					return nil
				})
				if err != nil {
					log.Errorf("Failed to execute the docker-manifest on worker %q: %v", workerAddress, err)
					return resource.RetryableError(errors.Wrapf(err, "failed to execute docker manifest on worker %s", workerAddress))
				}
				return nil
			})
			if err != nil {
				return errors.Wrapf(err, "error executing docker-manifest command on worker %s", workerAddress)
			}
			log.Infof("Manifested image %q on worker %q", id, workerAddress)
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return diag.Errorf("failed to manifest image %s: %v", id, err)
	}
	log.Infof("==== %s manifested on the highest worker ====", id)

	return nil
}

func resourceWindbagImageUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangeExcept("push") {
		d.SetId("")
		return resourceWindbagImageCreate(ctx, d, meta) // recreate
	}
	return resourceWindbagImageRead(ctx, d, meta)
}

func resourceWindbagImageDelete(_ context.Context, d *schema.ResourceData, _ interface{}) diag.Diagnostics {
	d.SetId("")
	return nil
}

func resourceWindbagImageID(image string) string {
	var img = docker.ParseImage(image)
	return strings.SplitN(img.Repository, "/", 2)[1]
}

func validationWindbagImageWorkerAddress(i interface{}, k string) (warnings []string, errors []error) {
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

func (p *provider) dialWorkerBySSH(ctx context.Context, id string, address string, ssh map[string]interface{}, configureDocker bool) (w dial.Dialer, err error) {
	var opts dial.SSHOptions
	opts.Address = address
	opts.Username = utils.ToString(ssh["username"])
	opts.Password = utils.ToString(ssh["password"])
	if v := utils.ToString(ssh["key"]); v != "" {
		opts.KeyPEMBlockBytes = utils.UnsafeStringToBytes(v)
	}
	if v := utils.ToString(ssh["cert"]); v != "" {
		opts.CertPEMBlockBytes = utils.UnsafeStringToBytes(v)
	}
	opts.WithAgent = utils.ToBool(ssh["with_agent"])

	var dockerBuild = p.docker
	err = resource.RetryContext(ctx, utils.ToDuration(ssh["retry_timeout"], 10*time.Minute), func() (rerr *resource.RetryError) {
		var err error

		// dail
		w, err = dial.SSH(opts)
		if err != nil {
			log.Errorf("Failed to dail worker %q: %v", address, err)
			return resource.RetryableError(err)
		}
		defer func() {
			if rerr != nil && w != nil {
				_ = w.Close()
			}
		}()

		// configure docker
		if !configureDocker {
			return nil
		}
		// configure docker, and install docker if the version isn't matched.
		if dockerBuild != nil {
			err = w.PowerShell(ctx, nil, func(ctx context.Context, ps *powershell.PowerShell) error {
				var psc, err = ps.Commands()
				if err != nil {
					return errors.Wrap(err, "failed to setup interaction")
				}
				defer func() {
					if err := psc.Close(); err != nil {
						log.Errorf("Failed to close interaction: %v", err)
					}
				}()

				var command = template.TryRender(
					p.docker,
					`
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
`,
				)
				_, stderr, err := psc.Execute(ctx, address, command)
				if err != nil {
					return errors.Wrap(err, "failed to verify docker version")
				}
				if stderr != "" {
					return errors.Errorf("error verifing docker version: %s", stderr)
				}

				return nil
			})
			if err != nil {
				log.Errorf("Failed to execute docker version validation on worker %q: %v", address, err)
				return resource.RetryableError(errors.Wrapf(err, "failed to verify docker version on worker %s", address))
			}

			// NB(thxCode): there is not robust solution to confirm that
			// a fresh host has been installed the docker server and restarted,
			// so we paused for 10 seconds and then dail again.
			time.Sleep(10 * time.Second)
			dockerBuild = nil // to skip the docker version verification
			return resource.RetryableError(errors.New("retry again"))
		}
		// confirm whether the docker server is established.
		if p.docker != nil {
			err = w.PowerShell(ctx, nil, func(ctx context.Context, ps *powershell.PowerShell) error {
				var psc, err = ps.Commands()
				if err != nil {
					return errors.Wrap(err, "failed to setup interaction")
				}
				defer func() {
					if err := psc.Close(); err != nil {
						log.Errorf("Failed to close interaction: %v", err)
					}
				}()

				var command = `docker info --format '{{ .ServerVersion }}';`
				_, stderr, err := psc.Execute(ctx, address, command)
				if err != nil {
					return errors.Wrap(err, "failed to confirm the state of docker server")
				}
				if stderr != "" {
					return errors.Errorf("error confirming the state of docker server: %s", stderr)
				}

				return nil
			})
			if err != nil {
				log.Errorf("Failed to get docker info on worker %q: %v", address, err)
				time.Sleep(10 * time.Second)
				return resource.RetryableError(errors.Wrapf(err, "failed to get docker info on worker %s", address))
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	log.Infof("%s dialed worker %q via SSH", id, address)

	return w, nil
}
