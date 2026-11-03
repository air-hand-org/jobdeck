using System.Net;
using Microsoft.Extensions.Configuration;
using Aspire.Hosting;
using Aspire.Hosting.ApplicationModel;
using Aspire.Hosting.Go;
using Aspire.Hosting.Yarp;

namespace apphosts;

public sealed class JobWorkerBuilder(
    IDistributedApplicationBuilder builder
    , EndpointReference jobApiGrpcEndpoint
    , EndpointReference authGrpcEndpoint
    , EndpointReference demoContainerEndpoint
    , string workerClientId
    , string pubsubVerificationToken
    , string imageTag
    , string forwardProxyImageTag
    , string repoRoot
    , bool useLocal
)
{
    private readonly string? localAppDirectory = GetLocalAppDirectory(repoRoot, useLocal);

    public EndpointReference Build(int port, bool isDebug)
    {
        var forwardProxy = ForwardProxy(demoContainerEndpoint);
        var workerClientSecret = builder.AddParameter("job-worker-client-secret", secret: true);
        return localAppDirectory is null
            ? ContainerBackend(port, forwardProxy, workerClientSecret)
                .GetEndpoint("push", KnownNetworkIdentifiers.DefaultAspireContainerNetwork)
            : LocalBackend(port, isDebug, forwardProxy, workerClientSecret)
                .GetEndpoint("push", KnownNetworkIdentifiers.DefaultAspireContainerNetwork);
    }

    private IResourceBuilder<ContainerResource> ContainerBackend(
        int port
        , EndpointReference forwardProxyEndpoint
        , IResourceBuilder<ParameterResource> workerClientSecret
    )
    {
        return ConfigureBackend(builder.AddContainer(
            name: "job-worker",
            image: "ghcr.io/air-hand-org/jobdeck-job-worker",
            tag: imageTag
        )
        .WithArgs("-worker", "3")
        .WithHttpEndpoint(
            name: "push",
            port: port,
            targetPort: port,
            env: "PORT"
        ), forwardProxyEndpoint, workerClientSecret);
    }

    private IResourceBuilder<ExecutableResource> LocalBackend(
        int port,
        bool isDebug,
        EndpointReference forwardProxyEndpoint,
        IResourceBuilder<ParameterResource> workerClientSecret
    )
    {
        var backend = ConfigureBackend(builder.AddGoApp(
            name: "job-worker",
            appDirectory: localAppDirectory!,
            packagePath: "./cmd/worker"
        )
        .WithAppArgs("-worker", "3")
        .WithHttpEndpoint(
            name: "push",
            port: port,
            env: "PORT"
        ), forwardProxyEndpoint, workerClientSecret);
        if (isDebug)
        {
            backend.WithDelveServer(new DelveServerOptions
            {
                Port = 2346,
                AcceptMultiClient = true,
                ContinueOnStart = true,
                OnlySameUser = false
            });
        }
        return backend;
    }

    private EndpointReference ForwardProxy(EndpointReference demoContainerEndpoint)
    {
        var proxy = localAppDirectory is null
            ? ContainerForwardProxy(demoContainerEndpoint)
            : LocalForwardProxy(demoContainerEndpoint);
        return proxy.GetEndpoint("proxy");
    }

    private IResourceBuilder<ContainerResource> ContainerForwardProxy(EndpointReference demoContainerEndpoint)
    {
        return ConfigureForwardProxy(builder.AddContainer(
            name: "forward-proxy",
            image: "ghcr.io/air-hand-org/jobdeck-forward-proxy",
            tag: forwardProxyImageTag
        ), demoContainerEndpoint);
    }

    private IResourceBuilder<ContainerResource> LocalForwardProxy(EndpointReference demoContainerEndpoint)
    {
        return ConfigureForwardProxy(builder.AddDockerfile(
                name: "forward-proxy",
                contextPath: Path.Combine(localAppDirectory!, "containers", "forward-proxy")
            ), demoContainerEndpoint);
    }

    private IResourceBuilder<T> ConfigureBackend<T>(
        IResourceBuilder<T> backend,
        EndpointReference forwardProxyEndpoint,
        IResourceBuilder<ParameterResource> workerClientSecret
    ) where T : IResource, IResourceWithEnvironment
    {
        return backend
            .WithReference(jobApiGrpcEndpoint)
            .WithReference(authGrpcEndpoint)
            .WithReference(forwardProxyEndpoint)
            .WithEnvironment(context =>
            {
                context.EnvironmentVariables["JOB_API_GRPC_ENDPOINT"] = jobApiGrpcEndpoint.Property(EndpointProperty.Url);
                context.EnvironmentVariables["AUTH_GRPC_ENDPOINT"] = authGrpcEndpoint.Property(EndpointProperty.Url);
                context.EnvironmentVariables["AUTH_CLIENT_ID"] = workerClientId;
                context.EnvironmentVariables["AUTH_CLIENT_SECRET"] = workerClientSecret;
                context.EnvironmentVariables["WORKER_HTTP_PROXY"] = forwardProxyEndpoint.Property(EndpointProperty.Url);
                context.EnvironmentVariables["LOCAL_PUBSUB_VERIFICATION_TOKEN"] = pubsubVerificationToken;
            });
    }

    private IResourceBuilder<T> ConfigureForwardProxy<T>(
        IResourceBuilder<T> proxy,
        EndpointReference demoContainerEndpoint
    ) where T : IResource, IResourceWithEnvironment, IResourceWithEndpoints
    {
        return proxy
            .WithHttpEndpoint(
                name: "proxy",
                port: 3128,
                targetPort: 3128
            )
            .WithReference(demoContainerEndpoint)
            .WithEnvironment("SQUID_ALLOW_PRIVATE_DEMO", "true")
            .WithEnvironment(context =>
            {
                context.EnvironmentVariables["SQUID_ALLOWED_DEMO_HOST_AND_PORT"] = demoContainerEndpoint.Property(EndpointProperty.HostAndPort);
            });
    }

    private static string? GetLocalAppDirectory(string repoRoot, bool useLocal)
    {
        if (!useLocal)
        {
            return null;
        }
        var path = Path.Combine(repoRoot, "services", "job-worker");
        if (!Directory.Exists(path))
        {
            return null;
        }
        if (!File.Exists(Path.Combine(path, "go.mod")))
        {
            throw new InvalidOperationException(
                $"The local job-worker checkout at '{path}' does not contain a go.mod file."
            );
        }
        return path;
    }
}
