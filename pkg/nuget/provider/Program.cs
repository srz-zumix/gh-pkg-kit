using NuGet.Protocol.Plugins;
using NuGet.Versioning;

namespace CredentialProvider.GhPkgKit;

internal static class Program
{
    public static async Task<int> Main(string[] args)
    {
        if (!args.SequenceEqual(new[] { "-Plugin" }))
        {
            Console.Error.WriteLine("This credential provider is launched by NuGet with -Plugin");
            return args.Length == 0 ? 0 : 1;
        }

        using var shutdown = new CancellationTokenSource();
        Console.CancelKeyPress += (_, eventArgs) =>
        {
            eventArgs.Cancel = true;
            shutdown.Cancel();
        };
        var handlers = new RequestHandlers();
        var registrations = new Dictionary<MessageMethod, IRequestHandler>
        {
            { MessageMethod.Initialize, new RequestHandler(async (connection, request, response, cancellation) =>
                {
                    var initialize = MessageUtilities.DeserializePayload<InitializeRequest>(request);
                    connection.Options.SetRequestTimeout(initialize.RequestTimeout);
                    await response.SendResponseAsync(request, new InitializeResponse(MessageResponseCode.Success), cancellation);
                }) },
            { MessageMethod.SetLogLevel, new RequestHandler((connection, request, response, cancellation) =>
                response.SendResponseAsync(request, new SetLogLevelResponse(MessageResponseCode.Success), cancellation)) },
            { MessageMethod.SetCredentials, new RequestHandler((connection, request, response, cancellation) =>
                response.SendResponseAsync(request, new SetCredentialsResponse(MessageResponseCode.Success), cancellation)) },
            { MessageMethod.GetOperationClaims, new RequestHandler((connection, request, response, cancellation) =>
                {
                    var claims = MessageUtilities.DeserializePayload<GetOperationClaimsRequest>(request);
                    var operations = claims.PackageSourceRepository == null && claims.ServiceIndex == null
                        ? new[] { OperationClaim.Authentication } : Array.Empty<OperationClaim>();
                    return response.SendResponseAsync(request, new GetOperationClaimsResponse(operations), cancellation);
                }) },
            { MessageMethod.GetAuthenticationCredentials, new RequestHandler(async (connection, request, response, cancellation) =>
                {
                    using var progress = AutomaticProgressReporter.Create(connection, request, TimeSpan.FromTicks(connection.Options.RequestTimeout.Ticks / 2), cancellation);
                    var credentials = MessageUtilities.DeserializePayload<GetAuthenticationCredentialsRequest>(request);
                    var result = await GitHubCredentials.GetAsync(credentials.Uri, cancellation);
                    await response.SendResponseAsync(request, result, cancellation);
                }) }
        };
        foreach (var registration in registrations)
        {
            handlers.TryAdd(registration.Key, registration.Value);
        }
        try
        {
            var defaults = ConnectionOptions.CreateDefault();
            var version = new SemanticVersion(2, 0, 0);
            var options = new ConnectionOptions(version, version, defaults.HandshakeTimeout, defaults.RequestTimeout);
            using var plugin = await PluginFactory.CreateFromCurrentProcessAsync(handlers, options, shutdown.Token);
            var closed = new TaskCompletionSource(TaskCreationOptions.RunContinuationsAsynchronously);
            plugin.Closed += (_, _) => closed.TrySetResult();
            plugin.Connection.Faulted += (_, _) => closed.TrySetException(new InvalidOperationException("NuGet plugin connection failed"));
            await closed.Task.WaitAsync(shutdown.Token);
            return 0;
        }
        catch (OperationCanceledException)
        {
            return 0;
        }
        catch (Exception)
        {
            Console.Error.WriteLine("NuGet credential provider failed; check gh authentication and plugin configuration");
            return 1;
        }
    }
}

internal sealed class RequestHandler(Func<IConnection, Message, IResponseHandler, CancellationToken, Task> handle) : IRequestHandler
{
    public CancellationToken CancellationToken => CancellationToken.None;

    public Task HandleResponseAsync(IConnection connection, Message request, IResponseHandler responseHandler, CancellationToken cancellationToken)
        => handle(connection, request, responseHandler, cancellationToken);
}
