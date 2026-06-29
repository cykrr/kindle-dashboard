package main

import (
	"fmt"
)

// extractIconPNG pulls the highest-resolution icon (up to 256x256 "jumbo")
// associated with a .lnk or .url shortcut and writes it to outPath as PNG.
//
// .lnk  -> resolve IconLocation, else TargetPath, then jumbo-extract.
// .url  -> read IconFile=/IconIndex= (Epic/Steam write a cached .ico here),
//          else fall back to the shortcut path itself.
//
// Jumbo extraction uses the system image list (SHGetImageList + SHIL_JUMBO),
// the only path that yields 256px art; ExtractAssociatedIcon caps at 32px.
func extractIconPNG(srcPath, outPath string) error {
	script := fmt.Sprintf(iconExtractScript, psEscape(srcPath), psEscape(outPath))
	if err := runPowerShell(false, script); err != nil {
		return fmt.Errorf("icon extract %q: %w", srcPath, err)
	}
	return nil
}

// iconExtractScript is a PowerShell template; %s/%s are the (single-quoted-safe)
// source shortcut path and output PNG path. $ErrorActionPreference=Stop makes any
// failure bubble to a non-zero exit so the Go caller sees an error.
const iconExtractScript = `
$ErrorActionPreference = 'Stop'
$Src = '%s'
$Out = '%s'

Add-Type -AssemblyName System.Drawing

function Resolve-IconSource([string]$path) {
    $ext = [IO.Path]::GetExtension($path).ToLower()
    if ($ext -eq '.lnk') {
        $sh = New-Object -ComObject WScript.Shell
        $lnk = $sh.CreateShortcut($path)
        $loc = $lnk.IconLocation
        if ($loc -and ($loc -notmatch '^\s*,?\s*0?\s*$')) {
            $parts = $loc -split ','
            $f = $parts[0].Trim()
            if ($f -and (Test-Path $f)) { return $f }
        }
        if ($lnk.TargetPath -and (Test-Path $lnk.TargetPath)) { return $lnk.TargetPath }
        return $path
    }
    if ($ext -eq '.url') {
        $iconFile = $null
        foreach ($line in (Get-Content -LiteralPath $path)) {
            if ($line -match '^\s*IconFile\s*=\s*(.+?)\s*$') { $iconFile = $matches[1] }
        }
        if ($iconFile -and (Test-Path $iconFile)) { return $iconFile }
        return $path
    }
    return $path
}

$sig = @'
using System;
using System.Drawing;
using System.Runtime.InteropServices;

public class JumboIcon {
    private const int SHIL_JUMBO = 0x4;           // 256x256
    private const int SHIL_EXTRALARGE = 0x2;      // 48x48 fallback
    private const uint SHGFI_SYSICONINDEX = 0x4000;
    private const int ILD_TRANSPARENT = 0x1;

    [StructLayout(LayoutKind.Sequential, CharSet = CharSet.Unicode)]
    private struct SHFILEINFO {
        public IntPtr hIcon;
        public int iIcon;
        public uint dwAttributes;
        [MarshalAs(UnmanagedType.ByValTStr, SizeConst = 260)] public string szDisplayName;
        [MarshalAs(UnmanagedType.ByValTStr, SizeConst = 80)] public string szTypeName;
    }

    [DllImport("shell32.dll", CharSet = CharSet.Unicode)]
    private static extern IntPtr SHGetFileInfo(string pszPath, uint dwFileAttributes,
        ref SHFILEINFO psfi, uint cbFileInfo, uint uFlags);

    [DllImport("shell32.dll", EntryPoint = "#727")]
    private static extern int SHGetImageList(int iImageList, ref Guid riid, out IImageList ppv);

    [ComImport, Guid("46EB5926-582E-4017-9FDF-E8998DAA0950"),
     InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
    private interface IImageList {
        [PreserveSig] int Add(IntPtr i, IntPtr m, ref int pi);
        [PreserveSig] int ReplaceIcon(int i, IntPtr h, ref int pi);
        [PreserveSig] int SetOverlayImage(int i, int o);
        [PreserveSig] int Replace(int i, IntPtr im, IntPtr m);
        [PreserveSig] int AddMasked(IntPtr im, int c, ref int pi);
        [PreserveSig] int Draw(ref IntPtr p);
        [PreserveSig] int Remove(int i);
        [PreserveSig] int GetIcon(int i, int flags, ref IntPtr picon);
    }

    public static Bitmap Get(string path) {
        var shinfo = new SHFILEINFO();
        SHGetFileInfo(path, 0, ref shinfo, (uint)Marshal.SizeOf(shinfo), SHGFI_SYSICONINDEX);
        var iid = new Guid("46EB5926-582E-4017-9FDF-E8998DAA0950");
        IImageList iml;
        if (SHGetImageList(SHIL_JUMBO, ref iid, out iml) != 0 || iml == null)
            throw new Exception("SHGetImageList(JUMBO) failed");
        IntPtr hIcon = IntPtr.Zero;
        iml.GetIcon(shinfo.iIcon, ILD_TRANSPARENT, ref hIcon);
        if (hIcon == IntPtr.Zero) throw new Exception("GetIcon returned null");
        using (var ico = Icon.FromHandle(hIcon)) {
            return ico.ToBitmap();
        }
    }
}
'@
Add-Type -TypeDefinition $sig -ReferencedAssemblies System.Drawing

$iconSrc = Resolve-IconSource $Src
# SHGetFileInfo returns a generic icon for forward-slash paths (e.g. Riot's
# IconLocation), so normalize to backslashes before extraction.
$iconSrc = $iconSrc -replace '/', '\'
$bmp = [JumboIcon]::Get($iconSrc)
$bmp.Save($Out, [System.Drawing.Imaging.ImageFormat]::Png)
$bmp.Dispose()
Write-Output ("OK {0} -> {1} ({2}x{3})" -f $iconSrc, $Out, $bmp.Width, $bmp.Height)
`
