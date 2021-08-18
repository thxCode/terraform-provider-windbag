package docker

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
)

const (
	commandArg = "ARG "

	platformArgTargetPlatform = "TARGETPLATFORM"
	platformArgTargetOs       = "TARGETOS"
	platformArgTargetArch     = "TARGETARCH"
	platformArgTargetVariant  = "TARGETVARIANT"
)

func InjectTargetPlatformArgsToDockerfile(raw io.Reader, os, arch, variant string) io.Reader {
	// validate
	if os == "" || arch == "" {
		return raw
	}
	var platform = fmt.Sprintf("%s/%s", os, arch)

	var b bytes.Buffer
	var s = bufio.NewScanner(raw)
	for s.Scan() {
		var bs = s.Bytes()
		if bytes.Compare(bs[:len(commandArg)], []byte(commandArg)) == 0 {
			switch platformArg := strings.TrimSpace(string(bs[len(commandArg):])); platformArg {
			case platformArgTargetPlatform:
				bs = []byte(commandArg + platformArg + `="` + platform + `"`)
			case platformArgTargetOs:
				bs = []byte(commandArg + platformArg + `="` + os + `"`)
			case platformArgTargetArch:
				bs = []byte(commandArg + platformArg + `="` + arch + `"`)
			case platformArgTargetVariant:
				bs = []byte(commandArg + platformArg + `="` + variant + `"`)
			}
		}
		b.Write(bs)
		b.WriteByte('\n')
	}
	return &b
}
