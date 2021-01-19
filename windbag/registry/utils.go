package registry

import (
	"strings"

	"github.com/docker/docker/registry"
)

// NormalizeRegistryAddress normalizes the registry address with or without the http(s):// prefix.
func NormalizeRegistryAddress(address string) string {
	if address == "index.docker.io" || address == "docker.io" {
		return "https://index.docker.io/v1/"
	}
	if !strings.HasPrefix(address, "https://") && !strings.HasPrefix(address, "http://") {
		return "https://" + address
	}
	return address
}

// ConvertToHostname converts the registry address to a hostname.
func ConvertToHostname(url string) string {
	return registry.ConvertToHostname(url)
}
