ARG RELEASEID=1809
# NB(thxCode): automatic platform ARGs, ref to:
# - https://docs.docker.com/engine/reference/builder/#automatic-platform-args-in-the-global-scope
ARG TARGETPLATFORM
ARG TARGETOS
ARG TARGETARCH

# --
# -- construct builder,
# -- inspired from https://raw.githubusercontent.com/rancher/hyperkube/v1.18/Dockerfile.windows
# --

FROM mcr.microsoft.com/windows/servercore:${RELEASEID} as builder
MAINTAINER weijia.mwj "weijia.mwj@alibaba-inc.com"

# $ProgressPreference: https://github.com/PowerShell/PowerShell/issues/2138#issuecomment-251261324
SHELL ["powershell", "-Command", "$ErrorActionPreference = 'Stop'; $ProgressPreference = 'SilentlyContinue';"]

# --
# -- install 7zip utils
# --

RUN if (-not (Get-Command Expand-7Zip -ErrorAction Ignore)) { \
       Install-PackageProvider -Name NuGet -Force -Verbose; \
       Install-Module -Name 7Zip4Powershell -Repository PSGallery -Force -Verbose; \
       if(-not $?) { \
            Write-Error "Failed to install package"; \
            Exit 1; \
       } \
    }

# --
# -- download rancher-wins
# --

ENV WINS_VERSION v0.0.4
RUN $URL = ('https://github.com/rancher/wins/releases/download/{0}/wins.exe' -f $env:WINS_VERSION); \
    \
    Write-Host ('Downloading wins from {0} ...' -f $URL); \
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12; \
    Invoke-WebRequest -UseBasicParsing -OutFile c:\wins.exe -Uri $URL; \
    \
    Write-Host 'Complete.'

# --
# -- download kubectl
# --

ENV K8S_VERSION v1.20.0
RUN $URL = ('https://dl.k8s.io/{0}/kubernetes-client-windows-amd64.tar.gz' -f $env:K8S_VERSION); \
    \
    function Expand-GZip ($inFile, $outFile) { \
        $input = New-Object System.IO.FileStream $inFile, ([IO.FileMode]::Open), ([IO.FileAccess]::Read), ([IO.FileShare]::Read); \
        $output = New-Object System.IO.FileStream $outFile, ([IO.FileMode]::Create), ([IO.FileAccess]::Write), ([IO.FileShare]::None); \
        $gzipStream = New-Object System.IO.Compression.GzipStream $input, ([IO.Compression.CompressionMode]::Decompress); \
        try { \
            if (!$input -or !$output -or !$gzipStream) { \
                Write-Error "Failed to Unzip the archive"; \
                Exit 1; \
            } \
            $buffer = New-Object byte[](1024); \
            while ($true) { \
                $read = $gzipstream.Read($buffer, 0, 1024); \
                if ($read -le 0 ) { \
                    break; \
                } \
                $output.Write($buffer, 0, $read); \
            } \
        } finally { \
            $gzipStream.Close(); \
            $output.Close(); \
            $input.Close(); \
        } \
    }; \
    \
    Write-Host ('Downloading Kubernetes from {0} ...' -f $URL); \
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12; \
    Invoke-WebRequest -UseBasicParsing -OutFile c:\kubernetes.tar.gz -Uri $URL; \
    \
    Write-Host 'Expanding ...'; \
    Expand-GZip c:\kubernetes.tar.gz c:\kubernetes.tar; \
    Expand-7Zip c:\kubernetes.tar c:\; \
    \
    Write-Host 'Complete.'

# --
# -- download cni plugins
# --

ENV CNI_PLUGINS_VERSION v0.8.2
RUN $URL = ('https://github.com/containernetworking/plugins/releases/download/{0}/cni-plugins-windows-amd64-{0}.tgz' -f $env:CNI_PLUGINS_VERSION); \
    \
    function DeGZip-File ($inFile, $outFile) { \
        $input = New-Object System.IO.FileStream $inFile, ([IO.FileMode]::Open), ([IO.FileAccess]::Read), ([IO.FileShare]::Read); \
        $output = New-Object System.IO.FileStream $outFile, ([IO.FileMode]::Create), ([IO.FileAccess]::Write), ([IO.FileShare]::None); \
        $gzipStream = New-Object System.IO.Compression.GzipStream $input, ([IO.Compression.CompressionMode]::Decompress); \
        try { \
            if (!$input -or !$output -or !$gzipStream) { \
                Write-Error "Failed to Unzip the archive"; \
                Exit 1; \
            } \
            $buffer = New-Object byte[](1024); \
            while ($true) { \
                $read = $gzipstream.Read($buffer, 0, 1024); \
                if ($read -le 0 ) { \
                    break; \
                } \
                $output.Write($buffer, 0, $read); \
            } \
        } finally { \
            $gzipStream.Close(); \
            $output.Close(); \
            $input.Close(); \
        } \
    }; \
    \
    Write-Host ('Downloading cni plugins from {0} ...' -f $URL); \
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12; \
    Invoke-WebRequest -UseBasicParsing -OutFile c:\containernetworking-plugins.tarz -Uri $URL; \
    \
    Write-Host 'Expanding ...'; \
    DeGZip-File c:\containernetworking-plugins.tarz c:\containernetworking-plugins.tar; \
    Expand-7Zip c:\containernetworking-plugins.tar c:\containernetworking\bin\.; \
    \
    Write-Host 'Complete.'

# --
# -- download flanneld
# --

ENV FLANNEL_VERSION v0.13.0-aliyun.1
RUN $URL = ('https://github.com/thxCode/coreos-flannel/releases/download/{0}/flanneld.exe' -f $env:FLANNEL_VERSION); \
    \
    Write-Host ('Downloading flannel from {0} ...' -f $URL); \
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12; \
    Invoke-WebRequest -UseBasicParsing -OutFile c:\flanneld.exe -Uri $URL; \
    \
    Write-Host 'Complete.'

# --
# -- construct flannel
# --

FROM mcr.microsoft.com/powershell:nanoserver-${RELEASEID}
MAINTAINER weijia.mwj "weijia.mwj@alibaba-inc.com"

USER ContainerAdministrator

ENV BACKEND_NAME="l2bridge" \
    BACKEND_TYPE="host-gw" \
    CLUSTER_SERVER="" \
    CLUSTER_DOMAIN="cluster.local" \
    CLUSTER_DNS="" \
    CLUSTER_CIDR="" \
    CLUSTER_SERVICE_CIDR="" \
    NODE_NAME=""

# copy artifacts from builder
COPY --from=builder /Windows/System32/certoc.exe /Windows/System32/netapi32.dll /Windows/System32/
COPY --from=builder /wins.exe /kubernetes/client/bin/kubectl.exe /Windows/System32/
COPY --from=builder /containernetworking/bin/host-local.exe /containernetworking/bin/flannel.exe /containernetworking/bin/win-overlay.exe /containernetworking/bin/win-bridge.exe /opt/cni/bin/
COPY --from=builder /flanneld.exe /opt/bin/

# copy scripts from host
COPY entrypoint.ps1 /

ENTRYPOINT ["pwsh.exe", "-NoLogo"]
