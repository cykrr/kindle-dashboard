$sig='[DllImport("user32.dll")]public static extern void keybd_event(byte bVk, byte bScan, uint dwFlags, UIntPtr dwExtraInfo);'
$k=Add-Type -MemberDefinition $sig -Name Win32KeyEvent -Namespace Win32Functions -PassThru
$k::keybd_event(0x08,0,0,[UIntPtr]::Zero)
$k::keybd_event(0x0D,0,0,[UIntPtr]::Zero)
Start-Sleep -Milliseconds 50
$k::keybd_event(0x08,0,2,[UIntPtr]::Zero)
$k::keybd_event(0x0D,0,2,[UIntPtr]::Zero)
