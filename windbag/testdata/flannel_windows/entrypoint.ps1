$ErrorActionPreference = 'Stop'

function Log-Debug
{
  $level = Get-Env -Key "LOG_LEVEL" -DefaultValue "debug"
  if ($level -ne "debug") {
    return
  }

  Write-Host -NoNewline -ForegroundColor White "DEBU: "
  $args | ForEach-Object {
    $arg = $_
    Write-Host -ForegroundColor Gray ("{0,-44}" -f $arg)
  }
}

function Log-Info
{
  $level = Get-Env -Key "LOG_LEVEL" -DefaultValue "debug"
  if (($level -ne "debug") -and ($level -ne "info")) {
    return
  }

  Write-Host -NoNewline -ForegroundColor Blue "INFO: "
  $args |ForEach-Object {
    $arg = $_
    Write-Host -ForegroundColor Gray ("{0,-44}" -f $arg)
  }
}

function Log-Warn
{
  Write-Host -NoNewline -ForegroundColor DarkYellow "WARN: "
  $args | ForEach-Object {
    $arg = $_
    Write-Host -ForegroundColor Gray ("{0,-44}" -f $arg)
  }
}

function Log-Error
{
  Write-Host -NoNewline -ForegroundColor DarkRed "ERRO: "
  $args | ForEach-Object {
    $arg = $_
    Write-Host -ForegroundColor Gray ("{0,-44}" -f $arg)
  }
}

function Log-Fatal
{
  Write-Host -NoNewline -ForegroundColor DarkRed "FATA: "
  $args | ForEach-Object {
    $arg = $_
    Write-Host -ForegroundColor Gray ("{0,-44}" -f $arg)
  }
  throw "PANIC"
}

function Get-Env
{
  param(
    [parameter(Mandatory = $true, ValueFromPipeline = $true)] [string]$Key,
    [parameter(Mandatory = $false)] [string]$DefaultValue = ""
  )

  try {
    $val = [Environment]::GetEnvironmentVariable($Key, [EnvironmentVariableTarget]::Process)
    if ($val) {
      return $val
    }
  } catch {}
  try {
    $val = [Environment]::GetEnvironmentVariable($Key, [EnvironmentVariableTarget]::User)
    if ($val) {
      return $val
    }
  } catch {}
  try {
    $val = [Environment]::GetEnvironmentVariable($Key, [EnvironmentVariableTarget]::Machine)
    if ($val) {
      return $val
    }
  } catch {}
  return $DefaultValue
}

function ConvertTo-Hashtable
{
  param (
    [parameter(Mandatory = $true, ValueFromPipeline = $true)] [PSCustomObject]$InputObject
  )

  if ($InputObject -is [array]) {
    foreach ($item in $value) {
      $item | ConvertTo-Hashtable
    }
  }

  if ($InputObject -is [hashtable] -or $InputObject -is [System.Collections.Specialized.OrderedDictionary]) {
    return $InputObject
  }

  $hash = [ordered]@{}
  if ($InputObject -is [System.Management.Automation.PSCustomObject]) {
    foreach ($prop in $InputObject.psobject.Properties) {
      $name = $prop.Name
      $value = $prop.Value

      if ($value -is [System.Management.Automation.PSCustomObject]) {
        $value = $value | ConvertTo-Hashtable
      }

      if ($value -is [array]) {
        $hashValue = @()
        if ($value[0] -is [hashtable] -or $value[0] -is [System.Collections.Specialized.OrderedDictionary] -or $value[0] -is [PSCustomObject]) {
          foreach ($item in $value) {
              $hashValue += ($item | ConvertTo-Hashtable)
          }
        } else {
          $hashValue = $value
        }
        $value = $hashValue
      }
      $hash.Add($name,$value)
    }
  }
  return $hash
}

function Create-Directory
{
  param (
    [parameter(Mandatory = $false, ValueFromPipeline = $true)] [string]$Path
  )

  if (Test-Path -Path $Path) {
    if (-not (Test-Path -Path $Path -PathType Container)) {
      # clean the same path file
      Remove-Item -Force -Path $Path -ErrorAction Ignore | Out-Null
    } else {
      return
    }
  }
  New-Item -Force -ItemType Directory -Path $Path | Out-Null
}

function Exist-File
{
    param (
      [parameter(Mandatory = $false, ValueFromPipeline = $true)] [string]$Path
    )

    if (Test-Path -Path $Path) {
        if (Test-Path -Path $Path -PathType Leaf) {
            return $true
        }
        # clean the same path directory
        Remove-Item -Recurse -Force -Path $Path -ErrorAction Ignore | Out-Null
    }
    return $false
}

function Transfer-File
{
  param (
    [parameter(Mandatory = $true)] [string]$Src,
    [parameter(Mandatory = $true)] [string]$Dst
  )

  if (Test-Path -PathType leaf -Path $Dst) {
    $dst_hasher = Get-FileHash -Path $Dst
    $src_hasher = Get-FileHash -Path $Src
    if ($dst_hasher.Hash -eq $src_hasher.Hash) {
      return
    }
  }

  try {
    $null = Copy-Item -Force -Path $Src -Destination $Dst
  } catch {
    throw "Could not transfer file $Src to $Dst : $($_.Exception.Message)"
  }
}

# render flanneld config
$flannel_config = @{}
if (Exist-File -Path "c:\etc\kube-flannel\config.conf") {
  $flannel_config = Get-Content -Path "c:\etc\kube-flannel\config.conf" | ConvertFrom-Json | ConvertTo-Hashtable
  Transfer-File -Src "c:\etc\kube-flannel\config.conf" -Dst "c:\host\etc\kube-flannel\config.conf"
} else {
  if (Exist-File -Path "c:\etc\kube-flannel\config.conf.tmpl") {
    $flannel_config = Get-Content -Path "c:\etc\kube-flannel\config.conf.tmpl" | ConvertFrom-Json | ConvertTo-Hashtable
    Log-Debug "Found flannel configuration template"
  }
  # Network
  if (-not $flannel_config["Network"]) {
    $flannel_config["Network"] = Get-Env -Key "CLUSTER_CIDR" -DefaultValue "PLEASE_SET_ENV_CLSUTER_CIDR"
  } else {
    $env_cluster_cidr = Get-Env -Key "CLUSTER_CIDR"
    if ($env_cluster_cidr) {
      $flannel_config["Network"] = $env_cluster_cidr
    }
  }
  # Backend
  $flannel_config_backend = $flannel_config["Backend"]
  if (-not $flannel_config_backend) {
    $flannel_config_backend = @{}
  }
  # Backend
  #   Name
  if (-not $flannel_config_backend["Name"]) {
    $flannel_config_backend["Name"] = Get-Env -Key "BACKEND_NAME" -DefaultValue "l2bridge"
  }
  # Backend
  #   Type
  if (-not $flannel_config_backend["Type"]) {
    $flannel_config_backend["Type"] = Get-Env -Key "BACKEND_TYPE" -DefaultValue "host-gw"
  }
  switch ($flannel_config_backend["Type"]) {
    'vxlan' {
      $flannel_config_backend["Port"] = 4789
      if (-not $flannel_config_backend["VNI"]) {
        $flannel_config_backend["VNI"] = 4096
      }
      if (-not $flannel_config_backend["MacPrefix"]) {
        $flannel_config_backend["MacPrefix"] = "0E-2A"
      }
    }
  }
  $flannel_config["Backend"] = $flannel_config_backend
  $flannel_config_conf = $flannel_config | ConvertTo-Json -Compress -Depth 32
  $flannel_config_conf | Out-File -NoNewline -Encoding utf8 -Force -FilePath "c:\host\etc\kube-flannel\config.conf"
  Log-Debug "Generated flannel configuration: $flannel_config_conf"
}

# render kubeconfig
if (Exist-File -Path "c:\etc\kube-flannel\kubeconfig.conf") {
  Transfer-File -Src "c:\etc\kube-flannel\kubeconfig.conf" -Dst "c:\host\etc\kube-flannel\kubeconfig.conf"
} else {
  # NB(thxCode): it's possible to get the default ep in-cluster?
  $cluster_server = Get-Env -Key "CLUSTER_SERVER"
  if (-not $cluster_server) {
    Log-Fatal "CLUSTER_SERVER environment variable is blank"
  }
  Transfer-File -Src "c:\var\run\secrets\kubernetes.io\serviceaccount\ca.crt" -Dst "c:\host\etc\kube-flannel\ca.crt"
  Transfer-File -Src "c:\var\run\secrets\kubernetes.io\serviceaccount\token" -Dst "c:\host\etc\kube-flannel\token"
  "apiVersion: v1
kind: Config
clusters:
- cluster:
    certificate-authority: c:/etc/kube-flannel/ca.crt
    server: ${cluster_server}
  name: default
contexts:
- context:
    cluster: default
    namespace: default
    user: default
  name: default
current-context: default
users:
- name: default
  user:
    tokenFile: c:/etc/kube-flannel/token" | Out-File -NoNewline -Encoding utf8 -Force -FilePath "c:\host\etc\kube-flannel\kubeconfig.conf"
}

# transfer artifacts for flannel
Transfer-File -Src "c:\opt\bin\flanneld.exe" -Dst "c:\host\opt\bin\flanneld.exe"

# render cni config
if (Exist-File -Path "c:\etc\cni\net.d\10-flannel.conf") {
  Transfer-File -Src "c:\etc\cni\net.d\10-flannel.conf" -Dst "c:\host\etc\cni\net.d\10-flannel.conf"
} else {
  $cni_config = @{}
  if (Exist-File -Path "c:\etc\cni\net.d\10-flannel.conf.tmpl") {
    $cni_config = Get-Content -Path "c:\etc\cni\net.d\10-flannel.conf.tmpl" | ConvertFrom-Json | ConvertTo-Hashtable
    Log-Debug "Found CNI configuration template"
  }
  # name
  $cni_config["name"] = "$($flannel_config["Backend"]["Name"])"
  # cniVersion
  if (-not $cni_config["cniVersion"]) {
    $cni_config["cniVersion"] = "0.3.0"
  }
  # type
  if (-not $cni_config["type"]) {
    $cni_config["type"] = "flannel"
  }
  # capabilities
  if (-not $cni_config["capabilities"]) {
    $cni_config["capabilities"] = @{
      dns = $true
    }
  }
  # delegate
  $cni_config_delegate = $cni_config["delegate"]
  if (-not $cni_config_delegate) {
    $cni_config_delegate = @{}
  }
  # delegate
  #   type
  switch ($flannel_config["Backend"]["Type"]) {
    'vxlan' {
      $cni_config_delegate["type"] = "win-overlay"
    }
    default {
      $cni_config_delegate["type"] = "win-bridge"
    }
  }
  # delegate
  #   dns
  $cni_config_delegate_dns = $cni_config_delegate["dns"]
  if (-not $cni_config_delegate_dns) {
    $cni_config_delegate_dns = @{}
  }
  # delegate
  #   dns
  #     nameservers
  $cni_config_delegate_dns_nameservers = $cni_config_delegate_dns["nameservers"]
  if ((-not $cni_config_delegate_dns_nameservers) -or ($cni_config_delegate_dns_nameservers.Length -eq 0)) {
    $dns = ""
    $env_service_cidr = Get-Env -Key "CLUSTER_SERVICE_CIDR" -DefaultValue "PLEASE_SET_ENV_CLUSTER_SERVICE_CIDR"
    if ($env_service_cidr) {
      $p = $env_service_cidr -split "\."
      if ($p.Length -eq 4) {
        $dns = ('{0}.{1}.{2}.10' -f $p[0],$p[1],$p[2])
        Log-Debug "Guessed DNS server is $dns"
      } else {
        $dns = "PLEASE_SET_ENV_CLUSTER_DNS"
      }
    }
    $env_dns_nameservers = Get-Env -Key "CLUSTER_DNS" -DefaultValue "$dns"
    if ($env_dns_nameservers) {
      $cni_config_delegate_dns_nameservers = @($env_dns_nameservers -split ",")
    }
  }
  $cni_config_delegate_dns["nameservers"] = $cni_config_delegate_dns_nameservers
  # delegate
  #   dns
  #     search
  $cni_config_delegate_dns_search = $cni_config_delegate_dns["search"]
  if ((-not $cni_config_delegate_dns_search) -or ($cni_config_delegate_dns_search.Length -eq 0)) {
    $env_dns_domain = Get-Env -Key "CLUSTER_DOMAIN" -DefaultValue "cluster.local"
    if ($env_dns_domain) {
      $domain = "svc.{0}" -f $env_dns_domain
      $cni_config_delegate_dns_search = @("$domain")
    }
  }
  $cni_config_delegate_dns["search"] = $cni_config_delegate_dns_search
  $cni_config_delegate["dns"] = $cni_config_delegate_dns
  # delegate
  #   policies
  $cni_config_delegate_policies = $cni_config_delegate["policies"]
  if (-not $cni_config_delegate_policies) {
    $env_cluster_cidr = $flannel_config["Network"]
    $env_service_cidr = Get-Env -Key "CLUSTER_SERVICE_CIDR" -DefaultValue "PLEASE_SET_ENV_CLUSTER_SERVICE_CIDR"
    switch ($cni_config_delegate["type"]) {
      'win-overlay' {
        $cni_config_delegate_policies = @(
          @{
            name = "EndpointPolicy"
            value = @{
              Type = "OutBoundNAT"
              ExceptionList = @(
                "$env_cluster_cidr"
                "$env_service_cidr"
              )
            }
          }
          @{
            name = "EndpointPolicy"
            value = @{
              Type = "ROUTE"
              NeedEncap = $true
              DestinationPrefix = "$env_service_cidr"
            }
          }
        )
      }
      default {
        $management_ip=""
        try {
          $bridge_net = "$(wins cli hns get-network --name="$($cni_config["name"])")"
          $bridge_net = $bridge_net | ConvertFrom-Json | ConvertTo-Hashtable
          $management_ip = $bridge_net["ManagementIP"]
        } catch {
          Log-Warn "Failed to get bridge net of $($cni_config["name"]), fallback to default network"
        }
        $host_net = "$(wins cli net get --address="$management_ip")"
        if (-not $?) {
          Log-Fatal "Failed to get host net $management_ip"
        }
        $host_net = $host_net | ConvertFrom-Json | ConvertTo-Hashtable

        $cni_config_delegate_policies = @(
          @{
            name = "EndpointPolicy"
            value = @{
              Type = "OutBoundNAT"
              ExceptionList = @(
                "$env_cluster_cidr"
                "$env_service_cidr"
                "$($host_net["SubnetCIDR"])"
              )
            }
          }
          @{
            name = "EndpointPolicy"
            value = @{
              Type = "ROUTE"
              NeedEncap = $true
              DestinationPrefix = "$env_service_cidr"
            }
          }
          @{
            name = "EndpointPolicy"
            value = @{
              Type = "ROUTE"
              NeedEncap = $true
              DestinationPrefix = "$($host_net["AddressCIDR"])"
            }
          }
        )
      }
    }
  }
  $cni_config_delegate["policies"] = $cni_config_delegate_policies
  $cni_config["delegate"] = $cni_config_delegate
  $cni_config_conf = $cni_config | ConvertTo-Json -Compress -Depth 32
  $cni_config_conf | Out-File -NoNewline -Encoding utf8 -Force -FilePath "c:\host\etc\cni\net.d\10-flannel.conf"
  Log-Debug "Generated CNI configuration: $cni_config_conf"
}

# transfer artifacts for cni
Transfer-File -Src "c:\opt\cni\bin\flannel.exe" -Dst "c:\host\opt\cni\bin\flannel.exe"
Transfer-File -Src "c:\opt\cni\bin\host-local.exe" -Dst "c:\host\opt\cni\bin\host-local.exe"
Transfer-File -Src "c:\opt\cni\bin\win-overlay.exe" -Dst "c:\host\opt\cni\bin\win-overlay.exe"
Transfer-File -Src "c:\opt\cni\bin\win-bridge.exe" -Dst "c:\host\opt\cni\bin\win-bridge.exe"

# run
$prc_path = "c:\opt\bin\flanneld.exe"
$prc_exposes = @(
  "UDP:4789"
)
$prc_args = @(
  "--ip-masq"
  "--iptables-forward-rules=false"
  "--kube-subnet-mgr"
  "--kubeconfig-file=c:\etc\kube-flannel\kubeconfig.conf"
  "--net-config-path=c:\etc\kube-flannel\config.conf"
)
try {
  $bridge_net = "$(wins cli hns get-network --name="$($cni_config["name"])")"
  $bridge_net = $bridge_net | ConvertFrom-Json | ConvertTo-Hashtable
  $prc_args += @(
    "--iface=$($bridge_net["ManagementIP"])"
  )
} catch {}
$env_node_name = Get-Env -Key "NODE_NAME"
if ($env_node_name) {
  $prc_envs = @(
    "NODE_NAME=$env_node_name"
  )
  wins cli prc run --path="$prc_path" --exposes="$($prc_exposes -join ' ')" --args="$($prc_args -join ' ')" --envs="$($prc_envs -join ' ')"
  return
}
wins cli prc run --path="$prc_path" --exposes="$($prc_exposes -join ' ')" --args="$($prc_args -join ' ')"
