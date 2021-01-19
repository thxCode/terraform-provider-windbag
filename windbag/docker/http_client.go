package docker

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

// getHTTPClientWithInsecure returns an http.Client instance skipped the insecure CA verification.
func getHTTPClientWithInsecure() *http.Client {
	var tr = getHTTPTransport()
	tr.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}

	return &http.Client{Transport: tr}
}

// getHTTPTransport returns an http.Transport instance as http.DefaultTransport.
func getHTTPTransport() *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}
