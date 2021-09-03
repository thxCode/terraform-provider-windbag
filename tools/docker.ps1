## https://docs.docker.com/install/windows/docker-ee/
$ErrorActionPreference = 'Stop'
$WarningPreference = 'SilentlyContinue'
$VerbosePreference = 'SilentlyContinue'
$DebugPreference = 'SilentlyContinue'
$ProgressPreference = 'SilentlyContinue';

function Get-VarEnv
{
    param(
        [parameter(Mandatory = $true)] [string]$Key,
        [parameter(Mandatory = $false)] [string]$DefaultValue = ""
    )

    try {
        $val = Get-Variable -Scope Global -Name $Key -ValueOnly -ErrorAction Ignore
        if ($val) {
            return $val
        }
    } catch {}
    try {
        $val = Get-Variable -Scope Local -Name $Key -ValueOnly -ErrorAction Ignore
        if ($val) {
            return $val
        }
    } catch {}
    try {
        $val = Get-Variable -Scope Private -Name $Key -ValueOnly -ErrorAction Ignore
        if ($val) {
            return $val
        }
    } catch {}
    try {
        $val = Get-Variable -Scope Script -Name $Key -ValueOnly -ErrorAction Ignore
        if ($val) {
            return $val
        }
    } catch {}

    return Get-Env -Key $Key -DefaultValue $DefaultValue
}

function Get-Env
{
    param(
        [parameter(Mandatory = $true)] [string]$Key,
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

function Set-Env
{
    param(
        [parameter(Mandatory = $true)] [string]$Key,
        [parameter(Mandatory = $false)] [string]$Value = ""
    )

    try {
        [Environment]::SetEnvironmentVariable($Key, $Value, [EnvironmentVariableTarget]::Process)
    } catch {}
    try {
        [Environment]::SetEnvironmentVariable($Key, $Value, [EnvironmentVariableTarget]::User)
    } catch {}
    try {
        [Environment]::SetEnvironmentVariable($Key, $Value, [EnvironmentVariableTarget]::Machine)
    } catch {}
}

function Log-Debug
{
    $level = Get-VarEnv -Key "LOG_LEVEL" -DefaultValue "debug"
    if ($level -ne "debug") {
        return
    }

    Write-Host -NoNewline -ForegroundColor White "DEBU: "
    $args | % {
        $arg = $_
        Write-Host -ForegroundColor Gray ("{0,-44}" -f $arg)
    }
}

function Log-Info
{
    $level = Get-VarEnv -Key "LOG_LEVEL" -DefaultValue "debug"
    if (($level -ne "debug") -and ($level -ne "info")) {
        return
    }

    Write-Host -NoNewline -ForegroundColor Blue "INFO: "
    $args | % {
        $arg = $_
        Write-Host -ForegroundColor Gray ("{0,-44}" -f $arg)
    }
}

function Log-Warn
{
    Write-Host -NoNewline -ForegroundColor DarkYellow "WARN: "
    $args | % {
        $arg = $_
        Write-Host -ForegroundColor Gray ("{0,-44}" -f $arg)
    }
}

function Log-Error
{
    Write-Host -NoNewline -ForegroundColor DarkRed "ERRO: "
    $args | % {
        $arg = $_
        Write-Host -ForegroundColor Gray ("{0,-44}" -f $arg)
    }
}

function Log-Fatal
{
    Write-Host -NoNewline -ForegroundColor DarkRed "FATA: "
    $args | % {
        $arg = $_
        Write-Host -ForegroundColor Gray ("{0,-44}" -f $arg)
    }

    throw "PANIC"
}

function Test-Command
{
    param (
        [parameter(Mandatory = $true, ValueFromPipeline = $true)] [string]$Command
    )

    try {
        if (Get-Command $Command) {
            return $true
        }
    } catch {
        return $false
    }
    return $false
}

function Add-MachineEnvironmentPath
{
    param (
        [parameter(Mandatory=$true)] [string]$Path
    )

    # Verify that the $Path is not already in the $env:Path variable.
    $pathForCompare = $Path.TrimEnd('\').ToLower()
    foreach ($p in $env:Path.Split(";")) {
        if ($p.TrimEnd('\').ToLower() -eq $pathForCompare) {
            return
        }
    }

    $newMachinePath = $Path + ";" + [System.Environment]::GetEnvironmentVariable("Path","Machine")
    [Environment]::SetEnvironmentVariable("Path", $newMachinePath, [System.EnvironmentVariableTarget]::Machine)
    $env:Path = $Path + ";" + $env:Path
}

function Execute-Binary
{
    param (
        [parameter(Mandatory = $true)] [string]$FilePath,
        [parameter(Mandatory = $false)] [string[]]$ArgumentList,
        [parameter(Mandatory = $false)] [string]$Encoding="Ascii"
    )

    $stdout = New-TemporaryFile
    $stderr = New-TemporaryFile
    $stdoutContent = ""
    $stderrContent = ""
    try {
        if ($ArgumentList) {
            Start-Process -NoNewWindow -Wait -FilePath $FilePath -ArgumentList $ArgumentList -RedirectStandardOutput $stdout.FullName -RedirectStandardError $stderr.FullName -ErrorAction Ignore
        } else {
            Start-Process -NoNewWindow -Wait -FilePath $FilePath -RedirectStandardOutput $stdout.FullName -RedirectStandardError $stderr.FullName -ErrorAction Ignore
        }
        $stdoutContent = Get-Content -Path $stdout.FullName -Encoding $Encoding
        $stderrContent = Get-Content -Path $stderr.FullName -Encoding $Encoding
    } catch {
        $stderrContent = $_.Exception.Message
    }
    $stdout.Delete()
    $stderr.Delete()

    $ret = ""
    if (-not [string]::IsNullOrEmpty($stdoutContent)) {
        $ret = $stdoutContent
    }
    if (-not [string]::IsNullOrEmpty($stderrContent)) {
        if ([string]::IsNullOrEmpty($ret)) {
            $ret = $stderrContent
        } else {
            $ret = $ret + "`n" + $stderrContent
        }
    }
    return $ret
}

function Test-Directory
{
  param (
    [parameter(Mandatory = $true, ValueFromPipeline = $true)] [string]$Path
  )
  return Test-Path -Path $Path -PathType Container
}

function Create-Directory
{
  param (
    [parameter(Mandatory = $true, ValueFromPipeline = $true)] [string]$Path
  )

  if (Test-Path -Path $Path) {
    if (Test-Directory -Path $Path) {
      return
    } else {
      Remove-Item -Force -Path $Path -ErrorAction Ignore | Out-Null
    }
  }
  New-Item -Force -ItemType Directory -Path $Path -ErrorAction Ignore | Out-Null
}

function Create-ParentDirectory
{
  param (
    [parameter(Mandatory = $true, ValueFromPipeline = $true)] [string]$Path
  )

  Create-Directory -Path (Split-Path -Path $Path) | Out-Null
}

function Compare-Semver
{
    param (
        [parameter(Mandatory = $true, ValueFromPipeline = $true)] [string]$Left,
        [parameter(Mandatory = $true, ValueFromPipeline = $true)] [string]$Right
    )

    try {
        $l = $Left -split "\."
        $r = $Right -split "\."
        $s = $l.Length
        if ($s -gt $r.Length) {
            $s = $r.Length
        }
        for ($i = 0; $i -lt $s; $i++) {
            $li = [int]($l[$i])
            $ri = [int]($r[$i])
            if ($li -lt $ri) {
                # Left < Right
                return 1
            }
            if ($li -gt $ri) {
                # Left > Right
                return -1
            }
        }
    } catch {}
    return 0
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

$DOCKER_VERSION = Get-VarEnv -Key "DOCKER_VERSION"
$DOCKER_DOWNLOAD_URI = Get-VarEnv -Key "DOCKER_DOWNLOAD_URI"
$DOCKER_CONFIGURATION_ALLOW_NONDISTRIBUTABLE_ARTIFACT = Get-VarEnv -Key "DOCKER_CONFIGURATION_ALLOW_NONDISTRIBUTABLE_ARTIFACT"
$DOCKER_CONFIGURATION_EXPERIMENTAL = Get-VarEnv -Key "DOCKER_CONFIGURATION_EXPERIMENTAL" -DefaultValue "true"
$DOCKER_CONFIGURATION_MAX_CONCURRENT_DOWNLOADS = Get-VarEnv -Key "DOCKER_CONFIGURATION_MAX_CONCURRENT_DOWNLOADS" -DefaultValue "8"
$DOCKER_CONFIGURATION_MAX_CONCURRENT_UPLOADS = Get-VarEnv -Key "DOCKER_CONFIGURATION_MAX_CONCURRENT_UPLOADS" -DefaultValue "8"
$DOCKER_CONFIGURATION_MAX_DOWNLOAD_ATTEMPTS = Get-VarEnv -Key "DOCKER_CONFIGURATION_MAX_DOWNLOAD_ATTEMPTS" -DefaultValue "10"
$DOCKER_CONFIGURATION_REGISTRY_MIRRORS = Get-VarEnv -Key "DOCKER_CONFIGURATION_REGISTRY_MIRRORS"

# validate
if ([string]::IsNullOrEmpty($DOCKER_VERSION)) {
    Log-Warn "Cannot verify Docker without version"
    exit 0
}

# install unpigz
if (-not (Test-Command -Command "unpigz")) {
    Invoke-WebRequest -UseBasicParsing -Uri "https://aliacs-k8s-cn-hongkong.oss-cn-hongkong.aliyuncs.com/public/pkg/windows/pigz/pigz-v2.3.1.zip" -OutFile "${tmp}\pigz.zip"
    Expand-Archive -Force -Path "${tmp}\pigz.zip" -DestinationPath "${env:ProgramFiles}"
    Add-MachineEnvironmentPath -Path "${env:ProgramFiles}\pigz"
    Add-MpPreference -ExclusionProcess "${env:ProgramFiles}\pigz\unpigz.exe" -ErrorAction Ignore
    Restart-Service -Name "docker" -Force -ErrorAction Ignore
}

# generate docker configuration
$dockerConfigurationPath = "${env:ProgramData}\docker\config\daemon.json"
$dockerConfiguration = Get-Content -Path "${dockerConfigurationPath}" -ErrorAction Ignore | ConvertFrom-Json | ConvertTo-Hashtable
if (-not $dockerConfiguration) {
    $dockerConfiguration = @{}
}
if (-not [string]::IsNullOrEmpty($DOCKER_CONFIGURATION_ALLOW_NONDISTRIBUTABLE_ARTIFACT)) {
    $dockerConfiguration["allow-nondistributable-artifacts"] = @($DOCKER_CONFIGURATION_ALLOW_NONDISTRIBUTABLE_ARTIFACT -split ",")
}
if ($DOCKER_CONFIGURATION_EXPERIMENTAL -eq "true") {
    $dockerConfiguration["experimental"] = $true
}
try {
    $dockerConfiguration["max-concurrent-downloads"] = [int]($DOCKER_CONFIGURATION_MAX_CONCURRENT_DOWNLOADS)
} catch {
    $dockerConfiguration["max-concurrent-downloads"] = 8
}
try {
    $dockerConfiguration["max-concurrent-uploads"] = [int]($DOCKER_CONFIGURATION_MAX_CONCURRENT_UPLOADS)
} catch{
    $dockerConfiguration["max-concurrent-uploads"] = 8
}
if ((Compare-Semver -Left "${DOCKER_VERSION}" -Right "19.03") -gt 0) {
    try {
        $dockerConfiguration["max-download-attempts"] = [int]($DOCKER_CONFIGURATION_MAX_DOWNLOAD_ATTEMPTS)
    } catch {
        $dockerConfiguration["max-download-attempts"] = 10
    }
}
if (-not [string]::IsNullOrEmpty($DOCKER_CONFIGURATION_REGISTRY_MIRRORS)) {
    $dockerConfiguration["registry-mirrors"] = @($DOCKER_CONFIGURATION_REGISTRY_MIRRORS -split ",")
}
Create-ParentDirectory -Path "${dockerConfigurationPath}"
$dockerConfiguration | ConvertTo-Json -Depth 32 -Compress | Out-File -FilePath "${dockerConfigurationPath}" -Encoding ascii -Force

# validate docker version
if (Test-Command -Command "dockerd") {
    $dockerVersionActualOutput = $(Execute-Binary -FilePath "dockerd" -ArgumentList @("--version"))
    $dockerVersionActual = "0"
    try {
        $dockerVersionActual = [regex]::matches($dockerVersionActualOutput, '^Docker version (?<Version>.*), build.*') | Foreach-Object { $_.Groups['Version'].Value }
    } catch {}
    $dockerVersionExpected = "${DOCKER_VERSION}"
    # Expected <= Actual
    if ((Compare-Semver -Left $dockerVersionExpected -Right $dockerVersionActual) -ge 0) {
        # start
        $service = Get-Service -Name "docker" -ErrorAction Ignore
        if (-not $service) {
            "$(dockerd --register-service --experimental)" | Out-Null
            $service = Get-Service -Name "docker" -ErrorAction Ignore
        }
        if (-not $service) {
            Log-Fatal "Found Docker daemon, but failed to register as a Windows Service"
        }
        $service | Where-Object {$_.StartType -ne "Automatic"} | Set-Service -StartupType Automatic | Out-Null
        $service | Where-Object {$_.Status -ne "Running"} | Start-Service -ErrorAction Ignore -WarningAction Ignore | Out-Null
        $service | Restart-Service | Out-Null

        Log-Info "Found Docker, version ${dockerVersionActual}"
        exit 0
    }
    Log-Warn "Found Docker, but the version ${dockerVersionActual} is stale"
}

# install docker
if ([string]::IsNullOrEmpty($DOCKER_DOWNLOAD_URI)) {
    $dockerIdxJson = $(curl.exe -sSkL https://dockermsft.blob.core.windows.net/dockercontainer/DockerMsftIndex.json | Out-String | ConvertFrom-Json)
    $vs = $DOCKER_VERSION -split '\.'
    switch ($vs.count) {
        3 {
            $dockerVersionJson = $dockerIdxJson | Select-Object -ErrorAction Ignore -ExpandProperty "versions" | Select-Object -ErrorAction Ignore -ExpandProperty "$DOCKER_VERSION"
            if (-not $dockerVersionJson) {
                Log-Fatal "Invalid Docker version: $DOCKER_VERSION, please view: https://dockermsft.blob.core.windows.net/dockercontainer/DockerMsftIndex.json"
            }
            $DOCKER_DOWNLOAD_URI = $dockerVersionJson.url
        }
        2 {
            $dockerVersionJson = $dockerIdxJson | Select-Object -ErrorAction Ignore -ExpandProperty "versions" | Select-Object -ErrorAction Ignore -ExpandProperty $($dockerIdxJson | Select-Object -ErrorAction Ignore -ExpandProperty "channels" | Select-Object -ErrorAction Ignore -ExpandProperty "$DOCKER_VERSION" | Select-Object -ErrorAction Ignore -ExpandProperty "version")
            if (-not $dockerVersionJson) {
                Log-Fatal "Invalid Docker version: $DOCKER_VERSION, please view: https://dockermsft.blob.core.windows.net/dockercontainer/DockerMsftIndex.json"
            }
            $DOCKER_DOWNLOAD_URI = $dockerVersionJson.url
        }
        default {
            if ($DOCKER_VERSION -eq "cs") {
                $dockerVersionJson = $dockerIdxJson | Select-Object -ErrorAction Ignore -ExpandProperty "versions" | Select-Object -ErrorAction Ignore -ExpandProperty $($dockerIdxJson | Select-Object -ErrorAction Ignore -ExpandProperty "channels" | Select-Object -ErrorAction Ignore -ExpandProperty $($dockerIdxJson.channels | Select-Object -ErrorAction Ignore -ExpandProperty "cs" | Select-Object -ErrorAction Ignore -ExpandProperty "alias") | Select-Object -ErrorAction Ignore -ExpandProperty "version")
                if (-not $dockerVersionJson) {
                    Log-Fatal "Could not find default Docker version, please indicate a specific version after viewing: https://dockermsft.blob.core.windows.net/dockercontainer/DockerMsftIndex.json"
                }
                $DOCKER_DOWNLOAD_URI = $dockerVersionJson.url
            } else {
                Log-Fatal "Invalid Docker version: $DOCKER_VERSION, please view: https://dockermsft.blob.core.windows.net/dockercontainer/DockerMsftIndex.json"
            }
        }
    }
}
Log-Info "Downloading Docker from $DOCKER_DOWNLOAD_URI ..."
Invoke-WebRequest -Uri "$DOCKER_DOWNLOAD_URI" -UseBasicParsing -OutFile "${env:TEMP}\docker.zip" | Out-Null

$service = Get-Service -Name "docker" -ErrorAction Ignore
if ($service) {
    Log-Warn "Stopping the stale Docker ..."
    Stop-Service -Name "docker" -Force -ErrorAction Ignore | Out-Null

    Log-Warn "Removing the stale Docker from Windows Service ..."
    if (Test-Command -Command "dockerd") {
        dockerd --unregister-service 2>&1 | Out-Null
    } else {
        sc.exe delete docker 2>&1 | Out-Null
    }
}

Log-Info "Expanding the Docker archive ..."
$removing = $true
# NB(thxCode): it seems like a bug on 1903,
# we cannot overwrite the binaries forcibly in one time.
# --- LEGACY_PROTECTION ---
while ($removing) {
    try {
        Expand-Archive -Path "${env:TEMP}\docker.zip" -DestinationPath "${env:ProgramFiles}" -Force -ErrorAction Ignore | Out-Null
        $removing = $false
    } catch {
        Log-Warn "Failed to override the stale Docker, try again."
        Start-Sleep -Seconds 5
    }
}
# --- LEGACY_PROTECTION ---
Remove-Item "${env:TEMP}\docker.zip" -Force | Out-Null

Log-Info "Refreshing the environment path with the Docker location ..."
Add-MachineEnvironmentPath -Path "${env:ProgramFiles}\docker"

Log-Info "Registering the Docker to Windows Service ..."
dockerd --register-service 2>&1 | Out-Null
$service = Get-Service -Name "docker" -ErrorAction Ignore
if (-not $service) {
    Log-Fatal "Failed to register Docker as a Windows Service"
}
$service | Where-Object {$_.StartType -ne "Automatic"} | Set-Service -StartupType Automatic | Out-Null

Log-Info "Verifying the required Windows Container Feature ..."
$installedType = Get-ComputerInfo | Select-Object -ErrorAction Ignore -ExpandProperty "WindowsInstallationType"
$restartNeeded = ""
if ($installedType -eq "Client") {
    try {
        $restartNeeded = Enable-WindowsOptionalFeature -Online -FeatureName "Containers" | Select-Object -ErrorAction Ignore -ExpandProperty "RestartNeeded"
        if ("${restartNeeded}" -eq "False") {
            $restartNeeded = "No"
        }
    } catch {}
} else {
    try {
        $restartNeeded = Install-WindowsFeature -Confirm:$false -Name "Containers" | Select-Object -ErrorAction Ignore -ExpandProperty "RestartNeeded"
    } catch {}
}
if ($restartNeeded -ne "No") {
    Log-Warn "Restart computer as installed Container Windows Feature ..."
    Restart-Computer
    exit 1
}
$service | Where-Object {$_.Status -ne "Running"} | Start-Service -ErrorAction Ignore -WarningAction Ignore | Out-Null
Log-Info "Docker version: $(docker info -f "{{ json .ServerVersion }}" 2>&1)"

Log-Info "Finished"
