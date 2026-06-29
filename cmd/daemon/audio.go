package main

import (
	"fmt"
	"net/http"
	"strconv"
)

// Audio control via Windows Core Audio, driven through PowerShell + embedded
// C# (same dependency-free pattern as icon extraction). Volume/mute use the
// documented IAudioEndpointVolume; default-output switching uses the
// undocumented-but-stable IPolicyConfig. Add-Type compiles on each call, so
// this is exposed on-demand via /audio — never on the SSE status poll.

const audioCSharp = `
using System;
using System.Collections.Generic;
using System.Runtime.InteropServices;

public class CoreAudio {
    [ComImport, Guid("BCDE0395-E52F-467C-8E3D-C4579291692E")] class MMDeviceEnumerator { }

    [Guid("A95664D2-9614-4F35-A746-DE8DB63617E6"), InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
    interface IMMDeviceEnumerator {
        int EnumAudioEndpoints(int dataFlow, int stateMask, out IMMDeviceCollection devices);
        int GetDefaultAudioEndpoint(int dataFlow, int role, out IMMDevice ppDevice);
        int GetDevice([MarshalAs(UnmanagedType.LPWStr)] string id, out IMMDevice dev);
        int RegisterEndpointNotificationCallback(IntPtr c);
        int UnregisterEndpointNotificationCallback(IntPtr c);
    }
    [Guid("0BD7A1BE-7A1A-44DB-8397-CC5392387B5E"), InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
    interface IMMDeviceCollection {
        int GetCount(out int count);
        int Item(int i, out IMMDevice dev);
    }
    [Guid("D666063F-1587-4E43-81F1-B948E807363F"), InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
    interface IMMDevice {
        int Activate(ref Guid iid, int ctx, IntPtr p, [MarshalAs(UnmanagedType.IUnknown)] out object iface);
        int OpenPropertyStore(int access, out IPropertyStore store);
        int GetId([MarshalAs(UnmanagedType.LPWStr)] out string id);
        int GetState(out int state);
    }
    [Guid("886d8eeb-8cf2-4446-8d02-cdba1dbdcf99"), InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
    interface IPropertyStore {
        int GetCount(out int c);
        int GetAt(int i, out PROPERTYKEY key);
        int GetValue(ref PROPERTYKEY key, out PROPVARIANT v);
        int SetValue(ref PROPERTYKEY key, ref PROPVARIANT v);
        int Commit();
    }
    [StructLayout(LayoutKind.Sequential)] struct PROPERTYKEY { public Guid fmtid; public int pid; }
    [StructLayout(LayoutKind.Explicit, Size = 24)] struct PROPVARIANT {
        [FieldOffset(0)] public short vt;
        [FieldOffset(8)] public IntPtr p;
    }

    [Guid("5CDF2C82-841E-4546-9722-0CF74078229A"), InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
    interface IAudioEndpointVolume {
        int RegisterControlChangeNotify(IntPtr n);
        int UnregisterControlChangeNotify(IntPtr n);
        int GetChannelCount(out int c);
        int SetMasterVolumeLevel(float level, ref Guid ctx);
        int SetMasterVolumeLevelScalar(float level, ref Guid ctx);
        int GetMasterVolumeLevel(out float level);
        int GetMasterVolumeLevelScalar(out float level);
        int SetChannelVolumeLevel(int ch, float level, ref Guid ctx);
        int SetChannelVolumeLevelScalar(int ch, float level, ref Guid ctx);
        int GetChannelVolumeLevel(int ch, out float level);
        int GetChannelVolumeLevelScalar(int ch, out float level);
        int SetMute(bool mute, ref Guid ctx);
        int GetMute(out bool mute);
    }

    [ComImport, Guid("870af99c-171d-4f9e-af0d-e63df40c2bc9")] class CPolicyConfigClient { }
    [Guid("f8679f50-850a-41cf-9c72-430f290290c8"), InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
    interface IPolicyConfig {
        int GetMixFormat([MarshalAs(UnmanagedType.LPWStr)] string id, IntPtr fmt);
        int GetDeviceFormat([MarshalAs(UnmanagedType.LPWStr)] string id, bool def, IntPtr fmt);
        int ResetDeviceFormat([MarshalAs(UnmanagedType.LPWStr)] string id);
        int SetDeviceFormat([MarshalAs(UnmanagedType.LPWStr)] string id, IntPtr a, IntPtr b);
        int GetProcessingPeriod([MarshalAs(UnmanagedType.LPWStr)] string id, bool def, IntPtr a, IntPtr b);
        int SetProcessingPeriod([MarshalAs(UnmanagedType.LPWStr)] string id, IntPtr p);
        int GetShareMode([MarshalAs(UnmanagedType.LPWStr)] string id, IntPtr m);
        int SetShareMode([MarshalAs(UnmanagedType.LPWStr)] string id, IntPtr m);
        int GetPropertyValue([MarshalAs(UnmanagedType.LPWStr)] string id, bool store, ref PROPERTYKEY key, out PROPVARIANT v);
        int SetPropertyValue([MarshalAs(UnmanagedType.LPWStr)] string id, bool store, ref PROPERTYKEY key, ref PROPVARIANT v);
        int SetDefaultEndpoint([MarshalAs(UnmanagedType.LPWStr)] string id, int role);
        int SetEndpointVisibility([MarshalAs(UnmanagedType.LPWStr)] string id, bool visible);
    }

    const int eRender = 0;
    const int DEVICE_STATE_ACTIVE = 1;
    const int eConsole = 0;
    static Guid IID_EndpointVolume = new Guid("5CDF2C82-841E-4546-9722-0CF74078229A");
    static PROPERTYKEY PKEY_Device_FriendlyName =
        new PROPERTYKEY { fmtid = new Guid("a45c254e-df1c-4efd-8020-67d146a850e0"), pid = 14 };

    static string FriendlyName(IMMDevice dev) {
        IPropertyStore store; dev.OpenPropertyStore(0, out store);
        PROPVARIANT pv; var key = PKEY_Device_FriendlyName;
        store.GetValue(ref key, out pv);
        return Marshal.PtrToStringUni(pv.p);
    }

    static IAudioEndpointVolume DefaultVol() {
        var en = (IMMDeviceEnumerator)(new MMDeviceEnumerator());
        IMMDevice dev; en.GetDefaultAudioEndpoint(eRender, eConsole, out dev);
        object o; var iid = IID_EndpointVolume;
        dev.Activate(ref iid, 1, IntPtr.Zero, out o);
        return (IAudioEndpointVolume)o;
    }

    public static int GetVolume() { float v; DefaultVol().GetMasterVolumeLevelScalar(out v); return (int)Math.Round(v * 100); }
    public static bool GetMute() { bool m; DefaultVol().GetMute(out m); return m; }
    public static void SetVolume(int pct) { var g = Guid.Empty; DefaultVol().SetMasterVolumeLevelScalar(pct / 100f, ref g); }
    public static void SetMute(bool m) { var g = Guid.Empty; DefaultVol().SetMute(m, ref g); }
    public static void ToggleMute() { SetMute(!GetMute()); }

    public class Dev { public string id; public string name; public bool isDefault; }

    public static List<Dev> ListOutputs() {
        var en = (IMMDeviceEnumerator)(new MMDeviceEnumerator());
        IMMDevice def; en.GetDefaultAudioEndpoint(eRender, eConsole, out def);
        string defId; def.GetId(out defId);
        IMMDeviceCollection col; en.EnumAudioEndpoints(eRender, DEVICE_STATE_ACTIVE, out col);
        int n; col.GetCount(out n);
        var list = new List<Dev>();
        for (int i = 0; i < n; i++) {
            IMMDevice d; col.Item(i, out d);
            string id; d.GetId(out id);
            list.Add(new Dev { id = id, name = FriendlyName(d), isDefault = (id == defId) });
        }
        return list;
    }

    public static void SetDefault(string id) {
        var pc = (IPolicyConfig)(new CPolicyConfigClient());
        pc.SetDefaultEndpoint(id, 0); // eConsole
        pc.SetDefaultEndpoint(id, 1); // eMultimedia
        pc.SetDefaultEndpoint(id, 2); // eCommunications
    }
}
`

// audioScript wraps the C# definition with a PowerShell tail that performs one
// operation.
func audioScript(tail string) string {
	return "$ErrorActionPreference='Stop'\n$sig=@'\n" + audioCSharp + "\n'@\nAdd-Type -TypeDefinition $sig\n" + tail
}

const audioStateTail = `
$outs = [CoreAudio]::ListOutputs() | ForEach-Object { @{ id = $_.id; name = $_.name; default = $_.isDefault } }
@{ volume = [CoreAudio]::GetVolume(); muted = [CoreAudio]::GetMute(); outputs = @($outs) } | ConvertTo-Json -Depth 4 -Compress
`

// audioStateJSON returns current volume/mute plus the list of output devices.
func audioStateJSON() (string, error) {
	return runPowerShellOut(audioScript(audioStateTail))
}

func audioSetVolume(arg string) error {
	n, err := strconv.Atoi(arg)
	if err != nil {
		return unknownActionError("volume_set: bad value " + arg)
	}
	if n < 0 {
		n = 0
	} else if n > 100 {
		n = 100
	}
	_, err = runPowerShellOut(audioScript(fmt.Sprintf("[CoreAudio]::SetVolume(%d)", n)))
	return err
}

func audioToggleMute() error {
	_, err := runPowerShellOut(audioScript("[CoreAudio]::ToggleMute()"))
	return err
}

func audioSetDefault(id string) error {
	if id == "" {
		return unknownActionError("audio_output: missing target")
	}
	_, err := runPowerShellOut(audioScript("[CoreAudio]::SetDefault('" + psEscape(id) + "')"))
	return err
}

// handleAudio serves GET /audio: current volume, mute, and output devices.
func handleAudio(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	if r.URL.Query().Get("key") != cfg.APIKey {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	j, err := audioStateJSON()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte(j))
}
