package docker

import (
	"fmt"
	"strings"

	"github.com/docker/docker/api/types"
)

// ConstructBuildCommand constructs the building command.
func ConstructBuildCommand(opts types.ImageBuildOptions, buildpath string) string {
	var sb strings.Builder
	sb.WriteString("docker build ")
	for k, v := range opts.BuildArgs {
		sb.WriteString(fmt.Sprintf("--build-arg %s=%s ", k, *v))
	}
	if opts.Dockerfile != "" {
		sb.WriteString(fmt.Sprintf("--file %s ", opts.Dockerfile))
	}
	if opts.ForceRemove {
		sb.WriteString("--force-rm ")
	}
	if !opts.Isolation.IsDefault() {
		sb.WriteString(fmt.Sprintf("--isolation %s ", opts.Isolation))
	}
	for k, v := range opts.Labels {
		sb.WriteString(fmt.Sprintf("--label %s=%s ", k, v))
	}
	if opts.NoCache {
		sb.WriteString("--no-cache ")
	}
	if opts.Remove {
		sb.WriteString("--rm ")
	}
	for _, v := range opts.Tags {
		sb.WriteString(fmt.Sprintf("--tag %s ", v))
	}
	if opts.Target != "" {
		sb.WriteString(fmt.Sprintf("--target %s ", opts.Target))
	}
	sb.WriteString(buildpath)
	return sb.String()
}

// ConstructImageInspectCommand constructs the inspecting image command.
func ConstructImageInspectCommand(tag string) string {
	var sb strings.Builder
	sb.WriteString("docker image inspect --format '{{json .}}' ")
	sb.WriteString(tag)
	return sb.String()
}

// ConstructImagePushCommand constructs the pushing image command.
func ConstructImagePushCommand(tag string) string {
	var sb strings.Builder
	sb.WriteString("docker push ")
	sb.WriteString(tag)
	return sb.String()
}

// ConstructManifestCreateCommand constructs the creating manifest command.
func ConstructManifestCreateCommand(tag string, manifests ...string) string {
	var sb strings.Builder
	sb.WriteString("$env:DOCKER_CLI_EXPERIMENTAL=\"enabled\"; docker manifest create --insecure --amend ")
	sb.WriteString(tag)
	sb.WriteString(" ")
	for idx := range manifests {
		sb.WriteString(manifests[idx])
		sb.WriteString(" ")
	}
	return sb.String()
}

// ConstructManifestPushCommand constructs the pushing manifest command.
func ConstructManifestPushCommand(tag string) string {
	var sb strings.Builder
	sb.WriteString("$env:DOCKER_CLI_EXPERIMENTAL=\"enabled\"; docker manifest push --purge ")
	sb.WriteString(tag)
	return sb.String()
}

// ConstructRegistryLoginCommand constructs the login registry command.
func ConstructRegistryLoginCommand(registry, username, password string) string {
	var sb strings.Builder
	sb.WriteString("docker login --username ")
	sb.WriteString(username)
	sb.WriteString(" --password ")
	sb.WriteString(password)
	sb.WriteString(" ")
	sb.WriteString(registry)
	return sb.String()
}
