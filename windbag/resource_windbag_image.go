package windbag

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"github.com/thxcode/terraform-provider-windbag/windbag/docker"
	"github.com/thxcode/terraform-provider-windbag/windbag/template"
	"github.com/thxcode/terraform-provider-windbag/windbag/utils"
	"github.com/thxcode/terraform-provider-windbag/windbag/worker"
	"github.com/thxcode/terraform-provider-windbag/windbag/worker/powershell"
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

		Schema: map[string]*schema.Schema{
			"build_arg": {
				Description: "Specify the build-time arguments.",
				Type:        schema.TypeMap,
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
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
				MinItems:    1,
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
			"build_worker": {
				Description: "Specify the workers to build.",
				Type:        schema.TypeSet,
				Required:    true,
				MinItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"address": {
							Description:  "Specify the address of worker.",
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
							Required:    true,
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
					},
				},
			},
		},
	}
}

func resourceWindbagImageCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var p = meta.(*provider)
	var id string
	var buildOpts = types.ImageBuildOptions{
		Version:   types.BuilderV1,
		BuildArgs: map[string]*string{},
		Labels:    map[string]string{},
	}

	// parse build options

	// tags
	buildOpts.Tags = utils.ToStringSlice(d.Get("tag"))
	id = resourceWindbagImageID(buildOpts.Tags[len(buildOpts.Tags)-1]) // use the last item as the resource ID
	// args
	if v, ok := d.GetOk("build_arg"); ok {
		for argName, vv := range v.(map[string]interface{}) {
			var argVal = vv.(string)
			buildOpts.BuildArgs[argName] = &argVal
		}
	}
	// labels
	if v, ok := d.GetOk("label"); ok {
		for labelName, vv := range v.(map[string]interface{}) {
			var labelVal = vv.(string)
			buildOpts.Labels[labelName] = labelVal
		}
	}
	// additions
	buildOpts.ForceRemove = utils.ToBool(d.Get("force_rm"))
	buildOpts.Isolation = container.Isolation(utils.ToString(d.Get("isolation")))
	buildOpts.NoCache = utils.ToBool(d.Get("no_cache"))
	buildOpts.Remove = utils.ToBool(d.Get("rm"))
	buildOpts.Target = utils.ToString(d.Get("target"))

	// construct build context

	// buildpath
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
	// dockerfile path
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

	// build

	log.Printf("[DEBUG] Building image %q\n", id)
	var buildWorkersInter = d.Get("build_worker").(*schema.Set).List()
	eg, egctx := errgroup.WithContext(ctx)
	for _, vv := range buildWorkersInter {
		var buildWorker = vv.(map[string]interface{})
		var workerAddress = utils.ToString(buildWorker["address"])
		var dialer, err = p.serviceSSHDialWorker(egctx, workerAddress, buildWorker["ssh"].(*schema.Set).List()[0].(map[string]interface{}))
		if err != nil {
			return diag.Errorf("failed to dial worker %s via SSH: %v", workerAddress, err)
		}

		var workerID = fmt.Sprintf("%s/%s", workerAddress, id)

		eg.Go(func() error {
			var workerWorkDir = utils.ToString(buildWorker["work_dir"])

			// create remote build context
			var err = dialer.PowerShell(egctx, nil, func(ctx context.Context, ps *powershell.PowerShell) error {
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
				return errors.Wrapf(err, "failed to craete remote build context of worker %s", workerAddress)
			}

			// transfer build path archive
			buildpathArchive, err := docker.GetBuildpathArchive(buildpath, dockerfilePath)
			if err != nil {
				return errors.Wrap(err, "failed to retrieve the buildpath")
			}
			var buildpathArchiveShippedDst = filepath.Join(workerWorkDir, "buildpath", fmt.Sprintf("%s.zip", id))
			_, err = dialer.Copy(egctx, buildpathArchive, buildpathArchiveShippedDst)
			if err != nil {
				return errors.Wrapf(err, "failed to ship the buildpath to worker %s", workerAddress)
			}

			// expand build path archive
			var buildpathArchiveExpandDst = filepath.Join(workerWorkDir, "buildpath", id)
			err = dialer.PowerShell(egctx, nil, func(ctx context.Context, ps *powershell.PowerShell) error {
				var psc, err = ps.Commands()
				if err != nil {
					return errors.Wrap(err, "failed to setup interaction")
				}
				defer func() {
					if err := psc.Close(); err != nil {
						log.Printf("[ERROR] Failed to close interaction: %v\n", err)
					}
				}()

				var command = template.TryRender(
					map[string]interface{}{
						"Src": buildpathArchiveShippedDst,
						"Dst": buildpathArchiveExpandDst,
					},
					`Expand-Archive -Force -Path "{{ .Src }}" -DestinationPath "{{ .Dst }}" | Out-Null`,
				)
				_, stderr, err := psc.Execute(ctx, workerID, command)
				if err != nil {
					return errors.Wrap(err, "failed to execute docker buildpath archive expansion")
				}
				if stderr != "" {
					return errors.Errorf("error executing docker buildpath archive expansion: %s", stderr)
				}

				return nil
			})
			if err != nil {
				return errors.Wrapf(err, "error executing expand-buildpath-archive command on worker %s", workerAddress)
			}

			// transfer build dockerfile
			var dockerfile, _ = os.Open(dockerfilePath)
			var dockerfileShippedDst = filepath.Join(workerWorkDir, "dockerfile", fmt.Sprintf("Dockerfile.%s", id))
			_, err = dialer.Copy(egctx, dockerfile, dockerfileShippedDst)
			if err != nil {
				return errors.Wrapf(err, "failed to ship the dockerfile to worker %s", workerAddress)
			}

			// docker build
			err = dialer.PowerShell(egctx, nil, func(ctx context.Context, ps *powershell.PowerShell) error {
				var psc, err = ps.Commands()
				if err != nil {
					return errors.Wrap(err, "failed to setup interaction")
				}
				defer func() {
					if err := psc.Close(); err != nil {
						log.Printf("[ERROR] Failed to close interaction: %v\n", err)
					}
				}()

				// get host release
				var command = `Get-ItemProperty -Path "HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion" | Select-Object -Property ReleaseId | ConvertTo-JSON -Compress;`
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
				var workerReleaseID = utils.ToString(hostVersion["ReleaseId"])

				// get host arch
				command = `[Environment]::GetEnvironmentVariable("PROCESSOR_ARCHITECTURE", [EnvironmentVariableTarget]::Machine);`
				stdout, stderr, err = psc.Execute(ctx, workerID, command)
				if err != nil {
					return errors.Wrap(err, "failed to retrieve host arch")
				}
				if stderr != "" {
					return errors.Errorf("error retrieving host arch: %s", stderr)
				}
				var workerArch = func() string {
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

				// docker build
				var workerOS = "windows"
				var workerPlatform = fmt.Sprintf("%s/%s", workerOS, workerArch)
				var workerTagSuffix = fmt.Sprintf("%s-%s-%s", workerOS, workerArch, workerReleaseID)
				command = func(opts types.ImageBuildOptions) string {
					// append build-args
					opts.BuildArgs["RELEASEID"] = &workerReleaseID
					opts.BuildArgs["TARGETPLATFORM"] = &workerPlatform
					opts.BuildArgs["TARGETOS"] = &workerOS
					opts.BuildArgs["TARGETARCH"] = &workerArch
					// redirect dockerfile
					opts.Dockerfile = dockerfileShippedDst
					// redirect tag
					for i := range opts.Tags {
						opts.Tags[i] = fmt.Sprintf("%s-%s", opts.Tags[i], workerTagSuffix)
					}
					return docker.ConstructBuildCommand(opts, buildpathArchiveExpandDst)
				}(buildOpts)
				_, stderr, err = psc.Execute(ctx, workerID, command)
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

			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return diag.Errorf("failed to build image %s: %v", id, err)
	}
	log.Printf("[INFO] Built image %q\n", id)

	d.SetId(id)
	return nil
}

func resourceWindbagImageRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var p = meta.(*provider)
	var id = d.Id()
	var tags = utils.ToStringSlice(d.Get("tag"))
	var buildWorkersInter = d.Get("build_worker").(*schema.Set).List()

	if !utils.ToBool(d.Get("push")) {
		log.Printf("[WARN] Skip to push the image %q", id)
		return nil
	}

	// login

	var registryLoginCommands = make(map[string]string)
	for _, tag := range tags {
		var img = docker.ParseImage(tag)
		var cred, ok = p.registryAuths[img.Registry]
		if !ok {
			log.Printf("[WARN] Cannot retrieve the credential of registry %q\n", img.Registry)
			continue
		}

		var command = docker.ConstructRegistryLoginCommand(img.Registry, cred.Username, cred.Password)
		registryLoginCommands[img.Registry] = command
	}
	if len(registryLoginCommands) == 0 {
		log.Printf("[WARN] There are not any registries want to login, you may fail to push any images")
	} else {
		var eg, egctx = errgroup.WithContext(ctx)
		for _, vv := range buildWorkersInter {
			var buildWorker = vv.(map[string]interface{})
			var workerAddress = utils.ToString(buildWorker["address"])
			var dialer, err = p.serviceSSHDialWorker(egctx, workerAddress, buildWorker["ssh"].(*schema.Set).List()[0].(map[string]interface{}))
			if err != nil {
				return diag.Errorf("failed to dial worker %s via SSH: %v", workerAddress, err)
			}

			var workerID = fmt.Sprintf("%s/%s", workerAddress, id)

			eg.Go(func() error {
				var err = dialer.PowerShell(egctx, nil, func(ctx context.Context, ps *powershell.PowerShell) error {
					var psc, err = ps.Commands()
					if err != nil {
						return errors.Wrap(err, "failed to setup interaction")
					}
					defer func() {
						if err := psc.Close(); err != nil {
							log.Printf("[ERROR] Failed to close interaction: %v\n", err)
						}
					}()

					for registry := range registryLoginCommands {
						log.Printf("[DEBUG] Logining registry %q on worker %q", registry, workerAddress)
						var command = registryLoginCommands[registry]
						var stdout, stderr, err = psc.Execute(ctx, workerID, command)
						if err != nil {
							return errors.Wrapf(err, "failed to login registry %s", registry)
						}
						if stderr != "" {
							if !strings.HasPrefix(stdout, "Login Succeeded") {
								return errors.Errorf("error loging registry %s: %s", registry, stderr)
							}
						}
						log.Printf("[INFO] Logon registry %q on worker %q", registry, workerAddress)
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

	// push

	log.Printf("[DEBUG] Pushing image %q\n", id)
	var eg, egctx = errgroup.WithContext(ctx)
	for _, vv := range buildWorkersInter {
		var buildWorker = vv.(map[string]interface{})
		var workerAddress = utils.ToString(buildWorker["address"])
		var dialer, err = p.serviceSSHDialWorker(egctx, workerAddress, buildWorker["ssh"].(*schema.Set).List()[0].(map[string]interface{}))
		if err != nil {
			return diag.Errorf("failed to dial worker %s via SSH: %v", workerAddress, err)
		}

		var workerID = fmt.Sprintf("%s/%s", workerAddress, id)

		eg.Go(func() error {
			var err = dialer.PowerShell(egctx, nil, func(ctx context.Context, ps *powershell.PowerShell) error {
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
				var command = `Get-ItemProperty -Path "HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion" | Select-Object -Property ReleaseId | ConvertTo-JSON -Compress;`
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
				var workerReleaseID = utils.ToString(hostVersion["ReleaseId"])

				// get host arch information
				command = `[Environment]::GetEnvironmentVariable("PROCESSOR_ARCHITECTURE", [EnvironmentVariableTarget]::Machine);`
				stdout, stderr, err = psc.Execute(ctx, workerID, command)
				if err != nil {
					return errors.Wrap(err, "failed to retrieve host arch")
				}
				if stderr != "" {
					return errors.Errorf("error retrieving host arch: %s", stderr)
				}
				var workerArch = func() string {
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

				// docker push
				var workerOS = "windows"
				var workerTagSuffix = fmt.Sprintf("%s-%s-%s", workerOS, workerArch, workerReleaseID)
				for _, tag := range tags {
					tag = fmt.Sprintf("%s-%s", tag, workerTagSuffix)
					log.Printf("[DEBUG] Pushing tag %q on worker %q", tag, workerAddress)
					command = docker.ConstructImagePushCommand(tag)
					_, stderr, err = psc.Execute(ctx, workerID, command)
					if err != nil {
						return errors.Wrapf(err, "failed to push tag %s", tag)
					}
					if stderr != "" {
						return errors.Errorf("error pushing tag %s: %s", tag, stderr)
					}
					log.Printf("[DEBUG] Pushed tag %q on worker %q", tag, workerAddress)
				}
				return nil
			})
			if err != nil {
				return errors.Wrapf(err, "error executing docker-push command on worker %s", workerAddress)
			}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return diag.Errorf("failed to push image %s: %v", id, err)
	}
	log.Printf("[INFO] Pushed image %q\n", id)

	return nil
}

func resourceWindbagImageUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	d.SetId("")
	return resourceWindbagImageCreate(ctx, d, meta) // recreate
}

func resourceWindbagImageDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	d.SetId("")
	return nil
}

func resourceWindbagImageID(image string) string {
	var img = docker.ParseImage(image)
	return strings.SplitN(img.Repository, "/", 2)[1]
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

func (p *provider) serviceGetImageDigest(ctx context.Context, image string) (string, error) {
	var opts []docker.GetImageDigestOption

	var si = docker.ParseImage(image)
	if ac, ok := p.registryAuths[si.Registry]; ok {
		opts = append(opts, docker.WithBasicAuth(ac.Username, ac.Password))
	}

	var digest, err = docker.GetImageDigest(ctx, image, append(opts, docker.WithManifestSupport())...)
	if err != nil {
		log.Printf("[WARN] Fallback to get image %s digest with manifest v1 specification: %v\n", image, err)
		return docker.GetImageDigest(ctx, image, append(opts, docker.WithManifestV1SupportOnly())...)
	}
	return digest, nil
}

func (p *provider) serviceSSHDialWorker(ctx context.Context, address string, ssh map[string]interface{}) (worker.Dialer, error) {
	var opts worker.DialSSHOptions
	opts.Address = utils.ToString(address)
	opts.Username = utils.ToString(ssh["username"])
	opts.Password = utils.ToString(ssh["password"])
	if v := utils.ToString(ssh["key"]); v != "" {
		opts.KeyPEMBlockBytes = utils.UnsafeStringToBytes(v)
	}
	if v := utils.ToString(ssh["cert"]); v != "" {
		opts.CertPEMBlockBytes = utils.UnsafeStringToBytes(v)
	}
	opts.WithAgent = utils.ToBool(ssh["with_agent"])

	log.Printf("[DEBUG] Dialing worker %q via SSH\n", address)
	var w, err = worker.DailSSH(opts)
	if err != nil {
		return nil, err
	}
	log.Printf("[DEBUG] Dialed worker %q via SSH\n", address)
	return w, nil
}
