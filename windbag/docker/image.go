package docker

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"

	"github.com/thxcode/terraform-provider-windbag/windbag/template"
	"github.com/thxcode/terraform-provider-windbag/windbag/utils"
)

// StructuredName structures the image name.
type StructuredName struct {
	Registry   string
	Repository string
	Tag        string
}

func (i StructuredName) String() string {
	return fmt.Sprintf("%s/%s:%s", i.Registry, i.Repository, i.Tag)
}

func (i StructuredName) GetManifestRequest(ctx context.Context) (*http.Request, error) {
	var v2API = template.TryRender(i, "https://{{ .Registry }}/v2/{{ .Repository }}/manifests/{{ .Tag }}")
	return http.NewRequestWithContext(ctx, http.MethodGet, v2API, nil)
}

// ParseImage parses the image string to a structure,
// it can parse the following image string:
// - docker.io/library/ubuntu:21.04 -> {docker.io, library/ubuntu, 21.04}
// - docker.io/library/ubuntu       -> {docker.io, library/ubuntu, latest}
// - library/ubuntu:20.10                 -> {docker.io, library/ubuntu, 20.10}
// - ubuntu:latest                        -> {docker.io, library/ubuntu, latest}
// - ubuntu                               -> {docker.io, library/ubuntu, latest}
func ParseImage(image string) StructuredName {
	var img StructuredName

	// tag
	if p := strings.LastIndex(image, ":"); p > 0 {
		img.Tag = image[p+1:]
		image = image[:p]
	} else {
		img.Tag = "latest"
	}

	var splits = strings.SplitN(image, "/", 3)
	switch len(splits) {
	case 3:
		img.Registry = splits[0]
		img.Repository = strings.Join([]string{splits[1], splits[2]}, "/")
	default:
		img.Registry = "docker.io"
		img.Repository = image
		if strings.Index(img.Repository, "/") == -1 {
			img.Repository = "library/" + img.Repository
		}
	}

	return img
}

// GetImageDigest returns the image digest.
func GetImageDigest(ctx context.Context, image string, opts ...GetImageDigestOption) (string, error) {
	var si = ParseImage(image)
	var req, err = si.GetManifestRequest(ctx)
	if err != nil {
		return "", errors.Wrap(err, "failed to create image manifest request")
	}

	var cli = getHTTPClientWithInsecure()
	for i := len(opts) - 1; i >= 0; i-- {
		cli.Transport = opts[i](cli.Transport)
	}

	resp, err := cli.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "failed to do image manifest request")
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// basic auth is valid or not needed
		return getDigestFromResponse(resp)
	case http.StatusUnauthorized:
		// either OAuth is required or the basic auth credential were invalid
		if strings.HasPrefix(resp.Header.Get("www-authenticate"), "Bearer") {
			var auth = parseAuthHeader(resp.Header.Get("www-authenticate"))
			var params = url.Values{}
			params.Set("service", auth["service"])
			params.Set("scope", auth["scope"])
			var tokenReq, err = http.NewRequestWithContext(ctx, "GET", auth["realm"]+"?"+params.Encode(), nil)
			if err != nil {
				return "", errors.Wrap(err, "failed to create registry token request")
			}

			tokenResp, err := cli.Do(tokenReq)
			if err != nil {
				return "", errors.Wrap(err, "failed to do registry token request")
			}
			defer tokenResp.Body.Close()

			if tokenResp.StatusCode != http.StatusOK {
				var bs, _ = ioutil.ReadAll(tokenResp.Body)
				return "", errors.Wrapf(err, "failed to do registry token request %d(%s): %s", tokenResp.StatusCode, tokenResp.Status, string(bs))
			}
			token, err := getTokenFromResponse(tokenResp)
			if err != nil {
				return "", err
			}

			req.Header.Set("Authorization", "Bearer "+token)
			digestResp, err := cli.Do(req)
			if err != nil {
				return "", errors.Wrap(err, "failed to do image manifest request")
			}
			defer digestResp.Body.Close()

			if digestResp.StatusCode != http.StatusOK {
				var bs, _ = ioutil.ReadAll(digestResp.Body)
				return "", errors.Errorf("requested image manifest, but got %d(%s): %s", digestResp.StatusCode, digestResp.Status, string(bs))
			}
			return getDigestFromResponse(digestResp)
		}
	}

	var bs, _ = ioutil.ReadAll(resp.Body)
	return "", errors.Errorf("requested image manifest, but got %d(%s): %s", resp.StatusCode, resp.Status, string(bs))
}

func parseAuthHeader(authenticate string) map[string]string {
	var opts = make(map[string]string)
	var parts = strings.SplitN(authenticate, " ", 2)
	parts = strings.Split(parts[1], ",")
	for idx := range parts {
		var item = strings.SplitN(parts[idx], "=", 2)
		var key = item[0]
		var val = strings.Trim(item[1], "\", ")
		opts[key] = val
	}
	return opts
}

func getTokenFromResponse(resp *http.Response) (string, error) {
	var body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "error reading token response body")
	}

	type tokenResponse struct {
		Token string `json:"token"`
	}
	var token tokenResponse
	if err := utils.UnmarshalJSON(body, &token); err != nil {
		return "", errors.Wrap(err, "error parsing token response body")
	}
	return token.Token, nil
}

func getDigestFromResponse(resp *http.Response) (string, error) {
	var header = resp.Header.Get("Docker-Content-Digest")
	if header == "" {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", errors.Wrap(err, "error reading digest response body")
		}
		return fmt.Sprintf("sha256:%x", sha256.Sum256(body)), nil
	}
	return header, nil
}
