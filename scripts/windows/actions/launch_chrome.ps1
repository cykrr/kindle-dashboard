$exe = 'chrome.exe'
$name = 'chrome'
$hwnd = (Get-Process $name -ErrorAction SilentlyContinue | Where-Object { $_.MainWindowHandle -ne 0 }).MainWindowHandle
if ($hwnd) {
    $sig = @'
[DllImport("user32.dll")] public static extern bool ShowWindowAsync(IntPtr hWnd, int nCmdShow);
[DllImport("user32.dll")] public static extern bool IsIconic(IntPtr hWnd);
[DllImport("user32.dll")] public static extern bool SetForegroundWindow(IntPtr hWnd);
'@
    $win32 = Add-Type -MemberDefinition $sig -Name Win32Toggle -Namespace Win32Functions -PassThru
    if ($win32::IsIconic($hwnd)) {
        $win32::ShowWindowAsync($hwnd, 9)
        $win32::SetForegroundWindow($hwnd)
    } else {
        $win32::ShowWindowAsync($hwnd, 6)
    }
} else {
    Start-Process $exe
}
