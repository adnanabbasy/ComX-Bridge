# ComX-Bridge C# Binding Guide

> [!NOTE]
> **Implementation Status**: The C API is implemented. The C# wrapper classes shown below are **reference implementations**. Official NuGet package is planned but not yet released.

This guide explains how to integrate ComX-Bridge in a .NET (C#) environment using **P/Invoke (Platform Invocation Services)**.

## 1. Overview

ComX-Bridge's `comx.dll` exposes a standard C interface, so native functions can be called directly using C#'s `[DllImport]` attribute.

*   **Requirements**: .NET 6.0+ recommended (or .NET Framework 4.7.2+)
*   **Platform**: Windows (x64) / Linux (x64)

## 2. P/Invoke Declaration (NativeMethods)

First, write a class mapping native functions to C# methods.

```csharp
using System;
using System.Runtime.InteropServices;

namespace ComxBridge
{
    internal static class NativeMethods
    {
        private const string LibName = "comx"; // Auto-detects comx.dll on Windows, libcomx.so on Linux

        [DllImport(LibName, CallingConvention = CallingConvention.Cdecl)]
        public static extern IntPtr comx_engine_create_with_config(string configJson);

        [DllImport(LibName, CallingConvention = CallingConvention.Cdecl)]
        public static extern void comx_engine_destroy(IntPtr engine);

        [DllImport(LibName, CallingConvention = CallingConvention.Cdecl)]
        public static extern int comx_engine_start(IntPtr engine);

        [DllImport(LibName, CallingConvention = CallingConvention.Cdecl)]
        public static extern int comx_engine_stop(IntPtr engine);

        [DllImport(LibName, CallingConvention = CallingConvention.Cdecl)]
        public static extern IntPtr comx_engine_get_gateway(IntPtr engine, string name);

        [DllImport(LibName, CallingConvention = CallingConvention.Cdecl)]
        public static extern int comx_gateway_send(IntPtr gateway, byte[] data, int len);

        [DllImport(LibName, CallingConvention = CallingConvention.Cdecl)]
        public static extern int comx_gateway_receive(IntPtr gateway, byte[] buffer, int maxLen, int timeoutMs);
    }
}
```

## 3. C# Wrapper Class (SafeHandle Recommended)

It is safer to create a wrapper class implementing the `IDisposable` pattern instead of handling `IntPtr` directly.

```csharp
using System;

namespace ComxBridge
{
    public class Engine : IDisposable
    {
        private IntPtr _handle;
        private bool _disposed;

        public Engine(string configJson)
        {
            _handle = NativeMethods.comx_engine_create_with_config(configJson);
            if (_handle == IntPtr.Zero) throw new Exception("Engine creation failed");
        }

        public void Start()
        {
            CheckDisposed();
            if (NativeMethods.comx_engine_start(_handle) != 0)
                throw new Exception("Failed to start engine");
        }

        public void Stop()
        {
            CheckDisposed();
            NativeMethods.comx_engine_stop(_handle);
        }

        public Gateway GetGateway(string name)
        {
            CheckDisposed();
            var gwHandle = NativeMethods.comx_engine_get_gateway(_handle, name);
            if (gwHandle == IntPtr.Zero) throw new Exception($"Gateway '{name}' not found");
            return new Gateway(gwHandle);
        }

        public void Dispose()
        {
            if (!_disposed)
            {
                if (_handle != IntPtr.Zero)
                {
                    NativeMethods.comx_engine_stop(_handle);
                    NativeMethods.comx_engine_destroy(_handle);
                    _handle = IntPtr.Zero;
                }
                _disposed = true;
            }
        }

        private void CheckDisposed()
        {
            if (_disposed) throw new ObjectDisposedException(nameof(Engine));
        }
    }

    public class Gateway
    {
        private readonly IntPtr _handle;

        internal Gateway(IntPtr handle) => _handle = handle;

        public void Send(byte[] data)
        {
            if (NativeMethods.comx_gateway_send(_handle, data, data.Length) != 0)
                throw new Exception("Send failed");
        }

        public byte[] Receive(int timeoutMs = 1000)
        {
            var buffer = new byte[4096];
            int received = NativeMethods.comx_gateway_receive(_handle, buffer, buffer.Length, timeoutMs);
            if (received < 0) throw new Exception("Receive error");
            if (received == 0) return null; // Timeout

            var result = new byte[received];
            Array.Copy(buffer, result, received);
            return result;
        }
    }
}
```

## 4. WPF/WinForms Usage Example

Example of UI thread integration.

```csharp
public partial class MainWindow : Window
{
    private ComxBridge.Engine _engine;

    public MainWindow()
    {
        InitializeComponent();
        
        string config = @"{ ""gateways"": [ ... ] }";
        _engine = new ComxBridge.Engine(config);
        _engine.Start();
    }

    private void btnSend_Click(object sender, RoutedEventArgs e)
    {
        try
        {
            var gw = _engine.GetGateway("my-device");
            gw.Send(new byte[] { 0x01, 0x02 });
            MessageBox.Show("Sent!");
        }
        catch (Exception ex)
        {
            MessageBox.Show($"Error: {ex.Message}");
        }
    }

    protected override void OnClosed(EventArgs e)
    {
        _engine.Dispose();
        base.OnClosed(e);
    }
}
```

## 5. Deployment Note (DLL Path)

The `comx.dll` file must exist in the same folder as the application executable (e.g., `bin/Debug`).
Set `comx.dll` to "Content" and "Copy to Output Directory" to "Copy if newer" in project properties.
