Get-PnpDevice | Where-Object { $_.InstanceId -like "*USB*" -and $_.Class -ne "USB" } | Select-Object Status,Class,FriendlyName,InstanceId | Format-Table -AutoSize
