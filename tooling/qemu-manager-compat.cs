// Launch current QEMU from Qemu Manager 7 while preserving its GUI settings.
using System;
using System.Collections.Generic;
using System.Diagnostics;
using System.IO;
using System.Linq;
using System.Runtime.InteropServices;
using System.Text;
using System.Web.Script.Serialization;
using System.Windows.Forms;

class QemuManagerCompat {
    static string Quote(string value) {
        var result = new StringBuilder("\"");
        int slashes = 0;
        foreach (char c in value) {
            if (c == '\\') { slashes++; continue; }
            result.Append('\\', c == '"' ? slashes * 2 + 1 : slashes);
            result.Append(c); slashes = 0;
        }
        return result.Append('\\', slashes * 2).Append('"').ToString();
    }
    static string Value(List<string> options, string key, string fallback) {
        string item = options.FirstOrDefault(s => s.StartsWith(key + "=", StringComparison.Ordinal));
        return item == null ? fallback : item.Substring(key.Length + 1);
    }
    static string Need(string[] args, ref int i) {
        if (++i >= args.Length) throw new ArgumentException("Missing value for " + args[i - 1]);
        return args[i];
    }
    static List<string> Translate(string[] args, string directory) {
        var output = new List<string>();
        var networks = new List<string>();
        var redirects = new List<string>();
        int usb = 0;
        bool localTime = false, rtcHack = false, rtcExplicit = false;
        for (int i = 0; i < args.Length; i++) {
            switch (args[i]) {
            case "-enable-kqemu": case "-kernel-kqemu": case "-no-kqemu":
                break;
            case "-localtime": localTime = true; break;
            case "-rtc-td-hack": rtcHack = true; break;
            case "-rtc":
                rtcExplicit = true; output.Add(args[i]); output.Add(Need(args, ref i)); break;
            case "-win2k-hack": break;
            case "-no-acpi": output.AddRange(new [] {"-machine", "acpi=off"}); break;
            case "-no-hpet": output.AddRange(new [] {"-machine", "hpet=off"}); break;
            case "-soundhw":
                foreach (string sound in Need(args, ref i).Split(',')) {
                    if (sound == "all" || sound == "pcspk")
                        throw new ArgumentException("Select a specific sound card in Qemu Manager instead of " + sound);
                    output.AddRange(new [] {"-device", sound == "ac97" ? "AC97" : sound});
                }
                break;
            case "-net": networks.Add(Need(args, ref i)); break;
            case "-redir":
                string rule = Need(args, ref i);
                string[] parts = rule.Split(':');
                if (parts.Length != 4 || (parts[0] != "tcp" && parts[0] != "udp"))
                    throw new ArgumentException("Unsupported old redirect: " + rule);
                redirects.Add("hostfwd=" + parts[0] + "::" + parts[1] + "-" + parts[2] + ":" + parts[3]);
                break;
            case "-usbdevice":
                string device = Need(args, ref i);
                if (device.StartsWith("disk:", StringComparison.Ordinal)) {
                    string id = "qm_usb" + usb++;
                    output.AddRange(new [] {"-drive", "if=none,id=" + id + ",format=raw,file=" + device.Substring(5),
                                           "-device", "usb-storage,drive=" + id});
                } else { output.AddRange(new [] {"-usbdevice", device}); }
                break;
            case "-L":
                string path = Need(args, ref i);
                if (String.Equals(Path.GetFullPath(path).TrimEnd('\\'), directory.TrimEnd('\\'),
                                  StringComparison.OrdinalIgnoreCase)) path = Path.Combine(directory, "share");
                output.AddRange(new [] {"-L", path}); break;
            default: output.Add(args[i]); break;
            }
        }
        if (!rtcExplicit && (localTime || rtcHack))
            output.AddRange(new [] {"-rtc", "base=" + (localTime ? "localtime" : "utc") + (rtcHack ? ",driftfix=slew" : "")});
        int number = 0, userNetwork = 0;
        foreach (string specification in networks) {
            var options = specification.Split(',').ToList();
            string kind = options[0]; options.RemoveAt(0);
            if (kind == "none") { output.AddRange(new [] {"-nic", "none"}); continue; }
            string vlan = Value(options, "vlan", "0");
            int hub;
            if (!Int32.TryParse(vlan, out hub) || hub < 0) throw new ArgumentException("Invalid network VLAN: " + vlan);
            options.RemoveAll(s => s.StartsWith("vlan=", StringComparison.Ordinal));
            string id = "qm_net" + number++;
            if (kind == "nic") {
                string model = Value(options, "model", "rtl8139"), mac = Value(options, "macaddr", "");
                string name = Value(options, "name", "");
                options.RemoveAll(s => s.StartsWith("model=") || s.StartsWith("macaddr=") || s.StartsWith("name="));
                output.AddRange(new [] {"-netdev", "hubport,id=" + id + ",hubid=" + hub,
                    "-device", model + ",netdev=" + id + (mac == "" ? "" : ",mac=" + mac) +
                    (name == "" ? "" : ",id=" + name) + (options.Count == 0 ? "" : "," + String.Join(",", options))});
            } else {
                if (kind == "user" && userNetwork++ == 0) options.AddRange(redirects);
                output.AddRange(new [] {"-netdev", kind + ",id=" + id +
                    (options.Count == 0 ? "" : "," + String.Join(",", options)),
                    "-netdev", "hubport,id=" + id + "_link,hubid=" + hub + ",netdev=" + id});
            }
        }
        if (redirects.Count != 0 && userNetwork == 0) throw new ArgumentException("Redirects need a user network backend");
        // The original manager's SDL display should remain a separate window.
        if (!args.Contains("-display") && !args.Contains("-nographic") && !args.Contains("-vnc"))
            output.AddRange(new [] {"-display", "sdl"});
        return output;
    }

    [StructLayout(LayoutKind.Sequential)] struct BasicLimits {
        public long ProcessTime, JobTime; public uint Flags;
        public UIntPtr MinWorkingSet, MaxWorkingSet; public uint ActiveProcesses;
        public UIntPtr Affinity; public uint Priority, Scheduling;
    }
    [StructLayout(LayoutKind.Sequential)] struct IoCounters {
        public ulong ReadOperations, WriteOperations, OtherOperations, ReadBytes, WriteBytes, OtherBytes;
    }
    [StructLayout(LayoutKind.Sequential)] struct ExtendedLimits {
        public BasicLimits Basic; public IoCounters Io;
        public UIntPtr ProcessMemory, JobMemory, PeakProcessMemory, PeakJobMemory;
    }
    [StructLayout(LayoutKind.Sequential, CharSet=CharSet.Unicode)] struct Startup {
        public uint Size; public string Reserved, Desktop, Title;
        public uint X, Y, Width, Height, XChars, YChars, Fill, Flags;
        public ushort Show, ReservedSize; public IntPtr ReservedPointer, Input, Output, Error;
    }
    [StructLayout(LayoutKind.Sequential)] struct ProcessInfo {
        public IntPtr Process, Thread; public uint ProcessId, ThreadId;
    }
    [DllImport("kernel32.dll", CharSet=CharSet.Unicode, SetLastError=true)]
    static extern IntPtr CreateJobObject(IntPtr security, string name);
    [DllImport("kernel32.dll", SetLastError=true)]
    static extern bool SetInformationJobObject(IntPtr job, int infoClass, ref ExtendedLimits limits, uint size);
    [DllImport("kernel32.dll", SetLastError=true)] static extern bool AssignProcessToJobObject(IntPtr job, IntPtr process);
    [DllImport("kernel32.dll", CharSet=CharSet.Unicode, SetLastError=true)]
    static extern bool CreateProcess(string application, StringBuilder command, IntPtr processSecurity,
        IntPtr threadSecurity, bool inherit, uint flags, IntPtr environment, string directory,
        ref Startup startup, out ProcessInfo process);
    [DllImport("kernel32.dll", SetLastError=true)] static extern uint ResumeThread(IntPtr thread);
    [DllImport("kernel32.dll")] static extern uint WaitForSingleObject(IntPtr handle, uint timeout);
    [DllImport("kernel32.dll")] static extern bool GetExitCodeProcess(IntPtr process, out uint code);
    [DllImport("kernel32.dll")] static extern bool TerminateProcess(IntPtr process, uint code);
    [DllImport("kernel32.dll")] static extern bool CloseHandle(IntPtr handle);
    static void Check(bool ok) { if (!ok) throw new System.ComponentModel.Win32Exception(Marshal.GetLastWin32Error()); }

    [STAThread] static int Main(string[] args) {
        string directory = Path.GetDirectoryName(System.Reflection.Assembly.GetExecutingAssembly().Location);
        try {
            if (args.Length > 1 && args[0] == "--translate-file") {
                string[] input = new JavaScriptSerializer().Deserialize<string[]>(File.ReadAllText(args[1]));
                File.WriteAllText(args[1] + ".translated.json", new JavaScriptSerializer().Serialize(Translate(input, directory)));
                return 0;
            }
            List<string> translated = Translate(args, directory);
            string backend = Path.Combine(directory, Path.GetFileNameWithoutExtension(
                System.Reflection.Assembly.GetExecutingAssembly().Location).Contains("x86_64") ?
                "qemu-system-x86_64-real.exe" : "qemu-system-i386.exe");
            if (!File.Exists(backend)) throw new FileNotFoundException("QEMU backend is missing", backend);
            IntPtr job = CreateJobObject(IntPtr.Zero, null); Check(job != IntPtr.Zero);
            var limits = new ExtendedLimits(); limits.Basic.Flags = 0x2000; // KILL_ON_JOB_CLOSE.
            ProcessInfo child = new ProcessInfo(); bool created = false;
            try {
                Check(SetInformationJobObject(job, 9, ref limits, (uint)Marshal.SizeOf(typeof(ExtendedLimits))));
                var startup = new Startup(); startup.Size = (uint)Marshal.SizeOf(typeof(Startup));
                string command = Quote(backend) + " " + String.Join(" ", translated.Select(Quote));
                Check(CreateProcess(backend, new StringBuilder(command), IntPtr.Zero, IntPtr.Zero,
                                    false, 4, IntPtr.Zero, directory, ref startup, out child));
                created = true;
                Check(AssignProcessToJobObject(job, child.Process));
                Check(ResumeThread(child.Thread) != UInt32.MaxValue);
                WaitForSingleObject(child.Process, UInt32.MaxValue);
                uint result; Check(GetExitCodeProcess(child.Process, out result)); return (int)result;
            } finally {
                CloseHandle(job);
                if (created) { TerminateProcess(child.Process, 1); CloseHandle(child.Thread); CloseHandle(child.Process); }
            }
        } catch (Exception error) {
            string log = Path.Combine(Environment.GetFolderPath(Environment.SpecialFolder.LocalApplicationData), "QemuManager-QEMU11-error.txt");
            File.WriteAllText(log, DateTime.Now + "\r\n" + error + "\r\n");
            MessageBox.Show(error.Message + "\r\nDetails: " + log, "Qemu Manager / QEMU 11", MessageBoxButtons.OK, MessageBoxIcon.Error);
            return 1;
        }
    }
}
