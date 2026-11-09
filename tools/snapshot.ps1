# Uso:  .\snapshot.ps1 antes        -> guarda estado antes de aplicar un tweak
#       (aplicas UN tweak en otra herramienta)
#       .\snapshot.ps1 despues      -> guarda estado y muestra la diferencia
# Ejecutar como Administrador. Resultado en scan\runs\<timestamp>\diff.txt
param([Parameter(Mandatory)][ValidateSet('antes','despues')]$fase, [string]$nombre = 'tweak')

$base = Join-Path $PSScriptRoot 'runs'
New-Item -ItemType Directory -Force $base | Out-Null
$state = Join-Path $base "$fase.txt"

function Dump-Key($k) {
    if (-not (Test-Path "Registry::$k")) { return }
    Get-ChildItem "Registry::$k" -Recurse -ErrorAction SilentlyContinue | ForEach-Object {
        $key = $_
        foreach ($n in $key.GetValueNames()) {
            $v = $key.GetValue($n)
            if ($v -is [byte[]]) { $v = ($v | ForEach-Object { '{0:x2}' -f $_ }) -join '' }
            elseif ($v -is [array]) { $v = $v -join ';' }
            "REG|$($key.Name)|$n|$v"
        }
    }
    $root = Get-Item "Registry::$k" -ErrorAction SilentlyContinue
    if ($root) { foreach ($n in $root.GetValueNames()) { "REG|$($root.Name)|$n|$($root.GetValue($n))" } }
}

$keys = @(
 'HKLM\SYSTEM\CurrentControlSet\Control\GraphicsDrivers',
 'HKLM\SYSTEM\CurrentControlSet\Control\PriorityControl',
 'HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Memory Management',
 'HKLM\SYSTEM\CurrentControlSet\Control\Power',
 'HKLM\SYSTEM\CurrentControlSet\Services\Tcpip\Parameters',
 'HKLM\SYSTEM\CurrentControlSet\Services\Tcpip\Parameters\Interfaces',
 'HKLM\SYSTEM\CurrentControlSet\Services\NDIS\Parameters',
 'HKLM\SYSTEM\CurrentControlSet\Services\LanmanServer\Parameters',
 'HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Multimedia',
 'HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Windows',
 'HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies',
 'HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Explorer',
 'HKLM\SOFTWARE\Policies\Microsoft\Windows',
 'HKLM\SOFTWARE\Policies\Microsoft\Windows Defender',
 'HKLM\SOFTWARE\Microsoft\Windows Defender',
 'HKLM\SOFTWARE\Microsoft\PolicyManager',
 'HKLM\SOFTWARE\NVIDIA Corporation',
 'HKCU\Software\Microsoft\Windows\CurrentVersion',
 'HKCU\Software\Microsoft\GameBar',
 'HKCU\System\GameConfigStore',
 'HKCU\Control Panel',
 'HKCU\Software\Microsoft\DirectX',
 'HKCU\Software\Microsoft\Windows NT\CurrentVersion\AppCompatFlags'
)

$out = New-Object System.Collections.Generic.List[string]
foreach ($k in $keys) { Dump-Key $k | ForEach-Object { $out.Add($_) } }
# Servicios (startmode/estado)
Get-CimInstance Win32_Service | ForEach-Object { $out.Add("SVC|$($_.Name)|$($_.StartMode)|$($_.State)") }
# Tareas programadas
Get-ScheduledTask -ErrorAction SilentlyContinue | ForEach-Object { $out.Add("TASK|$($_.TaskPath)$($_.TaskName)|$($_.State)") }
# powercfg / bcdedit / netsh / DNS
$out.Add("PWR|active|" + (powercfg /getactivescheme))
powercfg /query 2>$null | ForEach-Object { $out.Add("PWRQ|$_") }
bcdedit /enum '{current}' 2>$null | ForEach-Object { $out.Add("BCD|$_") }
netsh int tcp show global 2>$null | ForEach-Object { $out.Add("NETSH|$_") }
Get-DnsClientServerAddress -ErrorAction SilentlyContinue | ForEach-Object { $out.Add("DNS|$($_.InterfaceAlias)|$($_.AddressFamily)|$($_.ServerAddresses -join ',')") }
Get-NetAdapterAdvancedProperty -ErrorAction SilentlyContinue | ForEach-Object { $out.Add("NIC|$($_.Name)|$($_.DisplayName)|$($_.DisplayValue)") }

$out | Set-Content $state -Encoding utf8
Write-Host "Guardado $($out.Count) lineas en $state"

if ($fase -eq 'despues') {
    $antes = Join-Path $base 'antes.txt'
    if (-not (Test-Path $antes)) { throw 'Falta correr primero: .\snapshot.ps1 antes' }
    $a = Get-Content $antes; $d = Get-Content $state
    $diff = Compare-Object $a $d | Where-Object { $_.InputObject -notmatch 'LastWrite|Timestamp|Time\||Uptime' } |
        Sort-Object InputObject |
        ForEach-Object { if ($_.SideIndicator -eq '=>') { "+ $($_.InputObject)" } else { "- $($_.InputObject)" } }
    $dest = Join-Path $base ("diff_{0}_{1}.txt" -f $nombre, (Get-Date -Format yyyyMMdd_HHmmss))
    $diff | Set-Content $dest -Encoding utf8
    Write-Host "Cambios detectados: $($diff.Count)  ->  $dest"
    $diff | Select-Object -First 60
}

