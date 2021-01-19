package docker

import "net/http"

type GetImageDigestOption func(rt http.RoundTripper) http.RoundTripper

func WithBearerToken(token string) GetImageDigestOption {
	return func(rt http.RoundTripper) http.RoundTripper {
		return &bearerTokenRoundTripper{
			RoundTripper: rt,
			token:        token,
		}
	}
}

type bearerTokenRoundTripper struct {
	http.RoundTripper

	token string
}

func (rt *bearerTokenRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Header.Get("Authorization") == "" {
		if rt.token != "" {
			req.Header.Set("Authorization", "Bearer "+rt.token)
		}
	}
	return rt.RoundTripper.RoundTrip(req)
}

func WithBasicAuth(username, password string) GetImageDigestOption {
	return func(rt http.RoundTripper) http.RoundTripper {
		return &basicAuthRoundTripper{
			RoundTripper: rt,
			username:     username,
			password:     password,
		}
	}
}

type basicAuthRoundTripper struct {
	http.RoundTripper

	username string
	password string
}

func (rt *basicAuthRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Header.Get("Authorization") == "" {
		if rt.username != "" && rt.password != "" {
			req.SetBasicAuth(rt.username, rt.password)
		}
	}
	return rt.RoundTripper.RoundTrip(req)
}

func WithManifestSupport() GetImageDigestOption {
	return func(rt http.RoundTripper) http.RoundTripper {
		return &manifestV2SupportRoundTripper{
			RoundTripper: rt,
		}
	}
}

type manifestV2SupportRoundTripper struct {
	http.RoundTripper
}

func (rt *manifestV2SupportRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	var hasV1 bool
	for _, accept := range req.Header.Values("Accept") {
		if accept == "application/vnd.docker.distribution.manifest.v1+prettyjws" {
			hasV1 = true
			break
		}
	}
	if !hasV1 {
		req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v2+json")
		req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.list.v2+json")
		req.Header.Add("Accept", "application/vnd.oci.image.manifest.v1+json")
		req.Header.Add("Accept", "application/vnd.oci.image.index.v1+json")
	}
	return rt.RoundTripper.RoundTrip(req)
}

func WithManifestV1SupportOnly() GetImageDigestOption {
	return func(rt http.RoundTripper) http.RoundTripper {
		return &manifestV1SupportRoundTripper{
			RoundTripper: rt,
		}
	}
}

type manifestV1SupportRoundTripper struct {
	http.RoundTripper
}

func (rt *manifestV1SupportRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v1+prettyjws")
	return rt.RoundTripper.RoundTrip(req)
}
