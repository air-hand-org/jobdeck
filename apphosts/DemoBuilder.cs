using System.Net;
using Microsoft.Extensions.Configuration;
using Aspire.Hosting;
using Aspire.Hosting.ApplicationModel;
using Aspire.Hosting.Yarp;

namespace apphosts;

public sealed class DemoBuilder(
    IDistributedApplicationBuilder builder,
    string imageTag,
    string repoRoot,
    bool useLocal
)
{
    private readonly string? localAppDirectory = GetLocalAppDirectory(repoRoot, useLocal);

    public EndpointReference Build(int port, NetworkIdentifier networkId)
    {
        var backend = Backend();
        return ReverseProxy(port, backend, networkId);
    }

    private EndpointReference ReverseProxy(int port, EndpointReference backend, NetworkIdentifier networkId)
    {
        var yarp = builder.AddYarp("demo")
            .WithExternalHttpEndpoints()
            .WithHostHttpsPort(port)
            .WithConfiguration(yarp =>
            {
                yarp.AddRoute(backend);
            })
            ;
        return yarp.GetEndpoint("https", networkId);
    }

    private EndpointReference Backend()
    {
        return localAppDirectory is null
            ? ContainerBackend()
            : LocalBackend();
    }

    private EndpointReference ContainerBackend()
    {
        return builder.AddContainer(
            name: "demo-backend",
            image: "ghcr.io/air-hand-org/jobdeck-demo",
            tag: imageTag
        )
        .WithHttpEndpoint(
            name: "http",
            targetPort: 8080
        )
        .GetEndpoint("http");
    }

    private EndpointReference LocalBackend()
    {
        return builder.AddGoApp(
            name: "demo-backend",
            appDirectory: localAppDirectory!,
            packagePath: "./cmd/server"
        )
        .WithHttpEndpoint(
            name: "http",
            env: "PORT"
        )
        .GetEndpoint("http")
        ;
    }

    private static string? GetLocalAppDirectory(string repoRoot, bool useLocal)
    {
        if (!useLocal)
        {
            return null;
        }

        var path = Path.GetFullPath(Path.Combine(repoRoot, "..", "jobdeck-demo"));
        if (!Directory.Exists(path))
        {
            return null;
        }

        if (!File.Exists(Path.Combine(path, "go.mod")))
        {
            throw new InvalidOperationException(
                $"The local jobdeck-demo checkout at '{path}' does not contain a go.mod file."
            );
        }

        return path;
    }
}
