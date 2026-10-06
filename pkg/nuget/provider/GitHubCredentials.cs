using System.Diagnostics;
using System.Text.Json;
using NuGet.Protocol.Plugins;

namespace CredentialProvider.GhPkgKit;

internal static class GitHubCredentials
{
    internal static string? AuthenticationHost(Uri uri)
    {
        if (!uri.IsAbsoluteUri || uri.Scheme != Uri.UriSchemeHttps || uri.UserInfo.Length != 0)
        {
            return null;
        }
        var host = uri.IdnHost.ToLowerInvariant();
        if (host == "nuget.pkg.github.com")
        {
            return "github.com";
        }
        if (host.StartsWith("nuget.", StringComparison.Ordinal) && host.Length > "nuget.".Length)
        {
            return host["nuget.".Length..];
        }
        return uri.AbsolutePath.StartsWith("/_registry/nuget/", StringComparison.Ordinal) ? host : null;
    }

    internal static async Task<GetAuthenticationCredentialsResponse> GetAsync(Uri uri, CancellationToken cancellation)
    {
        var host = AuthenticationHost(uri);
        if (host == null)
        {
            return NotFound();
        }
        try
        {
            if (host != "github.com" && !string.Equals(Environment.GetEnvironmentVariable("GH_HOST"), host, StringComparison.OrdinalIgnoreCase))
            {
                var status = await RunGhAsync(new[] { "auth", "status", "--json", "hosts" }, cancellation);
                if (status.Output.Length == 0)
                {
                    return NotFound();
                }
                using var document = JsonDocument.Parse(status.Output);
                if (!document.RootElement.TryGetProperty("hosts", out var hosts) || !hosts.TryGetProperty(host, out _))
                {
                    return NotFound();
                }
            }
            var token = await RunGhAsync(new[] { "auth", "token", "--hostname", host }, cancellation);
            if (token.ExitCode != 0 || string.IsNullOrWhiteSpace(token.Output))
            {
                return NotFound("No gh auth token available; run gh auth login for the GitHub host");
            }
            return new GetAuthenticationCredentialsResponse("gh-pkg-kit", token.Output.Trim(), null, new[] { "basic" }, MessageResponseCode.Success);
        }
        catch (OperationCanceledException)
        {
            throw;
        }
        catch (Exception)
        {
            return NotFound("Could not retrieve credentials from gh; ensure GitHub CLI is installed and authenticated");
        }
    }

    private static GetAuthenticationCredentialsResponse NotFound(string? message = null)
        => new(null, null, message, null, MessageResponseCode.NotFound);

    private static async Task<(int ExitCode, string Output)> RunGhAsync(string[] arguments, CancellationToken cancellation)
    {
        var start = new ProcessStartInfo(Environment.GetEnvironmentVariable("GH_PATH") ?? "gh")
        {
            UseShellExecute = false,
            RedirectStandardInput = true,
            RedirectStandardOutput = true,
            RedirectStandardError = true,
            CreateNoWindow = true
        };
        foreach (var argument in arguments)
        {
            start.ArgumentList.Add(argument);
        }
        using var process = Process.Start(start) ?? throw new InvalidOperationException("Could not start gh");
        process.StandardInput.Close();
        using var registration = cancellation.Register(() =>
        {
            try
            {
                if (!process.HasExited) process.Kill(entireProcessTree: true);
            }
            catch (InvalidOperationException) { }
        });
        var output = process.StandardOutput.ReadToEndAsync(cancellation);
        var error = process.StandardError.ReadToEndAsync(cancellation);
        await process.WaitForExitAsync(cancellation);
        await error;
        return (process.ExitCode, await output);
    }
}
