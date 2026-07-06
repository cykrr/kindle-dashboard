$active = (powercfg /getactivescheme)
if ($active -match "High performance" -or $active -match "Ultimate Performance") {
    & "C:\Users\krr\bin\toggle-mode.ps1" -mode save
} else {
    & "C:\Users\krr\bin\toggle-mode.ps1" -mode power
}
