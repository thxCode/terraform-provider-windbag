## install guide: https://docs.microsoft.com/en-us/windows-server/administration/openssh/openssh_install_firstuse
## sshd_config guide: https://github.com/PowerShell/Win32-OpenSSH/wiki/sshd_config
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

function Judge
{
    param(
        [parameter(Mandatory = $true, ValueFromPipeline = $true)] [scriptBlock]$Block,
        [parameter(Mandatory = $false)] [int]$Timeout = 30,
        [parameter(Mandatory = $false)] [switch]$Reverse,
        [parameter(Mandatory = $false)] [switch]$Throw,
        [parameter(Mandatory = $false)] [string]$ErrorMessage = "Judge timeout"
    )

    $count = $Timeout
    while ($count -gt 0) {
        Start-Sleep -s 1

        if (&$Block) {
            if (-not $Reverse) {
                Start-Sleep -s 5
                break
            }
        } elseif ($Reverse) {
            Start-Sleep -s 5
            break
        }

        Start-Sleep -s 1
        $count -= 1
    }

    if ($count -le 0) {
        if ($Throw) {
            throw "$ErrorMessage"
        }

        Log-Fatal "$ErrorMessage"
    }
}

#
# disable
#

if ($(Get-VarEnv -Key "SSHD_ENABLED" -DefaultValue "enabled") -ne "enabled") {
    $sshd = Get-Service -Name "sshd" -ErrorAction Ignore
    if ($sshd) {
        if ($sshd.Status -eq 'Running') {
            Log-Warn "Stopping sshd ..."
            $sshd | Stop-Service -Force -ErrorAction Ignore | Out-Null
        }

        Log-Warn "Removing from Windows Service ..."
        sc.exe delete sshd | Out-Null

        Log-Warn "Shutting down firewall rule ..."
        Remove-NetFirewallRule -Name 'OpenSSH-Server-In-TCP' -ErrorAction Ignore | Out-Null
    }

    Log-Warn "Removing from Windows Capability ..."
    Get-WindowsCapability -Online -ErrorAction Ignore | ? Name -like 'OpenSSH.Server*' | Remove-WindowsCapability -Online -ErrorAction SilentlyContinue | Out-Null

    Log-Info "Finished"
    exit 0
}

#
# enable
#

$SSH_USER = Get-VarEnv -Key "SSH_USER"
$SSH_USER_PASSWORD = Get-VarEnv -Key "SSH_USER_PASSWORD"
$SSH_USER_PUBLICKEY = Get-VarEnv -Key "SSH_USER_PUBLICKEY"
$SSH_USER_GROUP = Get-VarEnv -Key "SSH_USER_GROUP"

$sshd = Get-Service -Name "sshd" -ErrorAction Ignore
if (-not $sshd) {
    Log-Info "Installing sshd ..."
    {
        try {
            Get-WindowsCapability -Online -ErrorAction Stop | ? Name -like 'OpenSSH.Server*' | Add-WindowsCapability -Online -ErrorAction Stop | Out-Null
            return $true
        } catch {
            return $false
        }
    } | Judge -ErrorMessage "Failed to enable sshd Windows Capability" -Timeout 60
}

$sshd = Get-Service -Name "sshd" -ErrorAction Ignore
if (-not $sshd) {
    Log-Fatal "Could not find sshd service"
}
if ($sshd.StartupType -ne 'Automatic') {
    Set-Service -Name "sshd" -StartupType 'Automatic' -ErrorAction Ignore
}
if ($sshd.Status -ne 'Running') {
    Log-Info "Starting sshd ..."
    {
        try {
            Start-Service -Name "sshd" | Out-Null
            return $true
        } catch {
            return $false
        }
    } | Judge -ErrorMessage "Failed to start sshd service" -Timeout 60
}

$sFirewall = Get-NetFirewallRule -Name *ssh*
if (-not $sFirewall) {
    $sFirewall = New-NetFirewallRule -Name "OpenSSH-Server-In-TCP" -DisplayName "OpenSSH Server (sshd)" -Enabled False -Direction Inbound -Protocol TCP -Action Allow -LocalPort 22
}
if ((-not $sFirewall.Enabled) -or ($sFirewall.Action -ne 'Allow')) {
    Log-Info "Enabling sshd firewall rule ..."
    $sFirewall | Set-NetFirewallRule -Enabled true -Action Allow
}

if ($SSH_USER) {
    $sUser = Get-LocalUser $SSH_USER -ErrorAction Ignore
    if (-not $sUser) {
        if (-not [string]::IsNullOrEmpty($SSH_USER_PASSWORD)) {
            Log-Info "Creating user $SSH_USER with password ..."
            $sUser = New-LocalUser -Name $SSH_USER `
                -Description "sshd-mgr created $SSH_USER" `
                -Password (ConvertTo-SecureString -AsPlainText $SSH_USER_PASSWORD -Force) `
                -PasswordNeverExpires `
                -AccountNeverExpires
        } else {
            Log-Info "Creating user $SSH_USER without password ..."
            $sUser = New-LocalUser -Name $SSH_USER `
                -Description "sshd-mgr created $SSH_USER" `
                -NoPassword `
                -AccountNeverExpires
        }
    }

    if (-not $SSH_USER_GROUP) {
        $SSH_USER_GROUP = "Administrators"
    }
    $sGroup = Get-LocalGroup $SSH_USER_GROUP -ErrorAction Ignore
    if (-not $sGroup) {
        Log-Info "Creating group $SSH_USER_GROUP ..."
        $sGroup = New-LocalGroup -Name $SSH_USER_GROUP `
            -Description "sshd-mgr created $SSH_USER_GROUP group"
    }

    Log-Info "Joining user $SSH_USER into $SSH_USER_GROUP group ..."
    Add-LocalGroupMember -Group $SSH_USER_GROUP -Member $SSH_USER -ErrorAction Ignore | Out-Null

    if (([string]::IsNullOrEmpty($SSH_USER_PASSWORD)) -and (-not [string]::IsNullOrEmpty($SSH_USER_PUBLICKEY))) {
        Log-Info "Locating the public key of user $SSH_USER ..."

        if ($SSH_USER_GROUP -eq "Administrators") {
            $sPath = "c:\ProgramData\ssh\administrators_authorized_keys"
            $SSH_USER_PUBLICKEY.Trim() + "`r`n" | Out-File -Encoding utf8 -Append -FilePath "$sPath"

            $fileExpectedAcl = "O:BAG:SYD:PAI(A;;FA;;;SY)(A;;FA;;;BA)"
            $sAcl = Get-ACL "$sPath"
            $sAcl.SetSecurityDescriptorSddlForm($fileExpectedAcl)
            $sAcl | Set-ACL $sPath
        } else {
            $sPath = "c:\Users\${ssh_user}\.ssh"
            New-Item -Force -Type "Directory" -Path $sPath | Out-Null
            $sPath = "$sPath\authorized_keys"
            $SSH_USER_PUBLICKEY.Trim() + "`r`n" | Out-File -Encoding utf8 -Append -FilePath "$sPath"
            $fileExpectedAcl = "O:BAG:SYD:PAI(A;;FA;;;SY)(A;;FA;;;$($sUser.SID.Value))"
            $sAcl = Get-ACL "$sPath"
            $sAcl.SetSecurityDescriptorSddlForm($fileExpectedAcl)
            $sAcl | Set-ACL $sPath

            $sPath = "c:\Windows\.ssh"
            New-Item -Force -Type "Directory" -Path $sPath | Out-Null
            $sPath = "$sPath\authorized_keys"
            $SSH_USER_PUBLICKEY.Trim() + "`r`n" | Out-File -Encoding utf8 -Append -FilePath "$sPath"
            $fileExpectedAcl = "O:BAG:SYD:PAI(A;;FA;;;SY)(A;;FA;;;BA)"
            $sAcl = Get-ACL "$sPath"
            $sAcl.SetSecurityDescriptorSddlForm($fileExpectedAcl)
            $sAcl | Set-ACL $sPath
        }
    }
}

Log-Info "Finished, and going to restart the computer immediatly"

Restart-Computer
