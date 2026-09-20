using System.Net;
using Microsoft.Extensions.Configuration;
using Aspire.Hosting;
using Aspire.Hosting.ApplicationModel;
using Aspire.Hosting.Go;
using Aspire.Hosting.Yarp;

namespace apphosts;

public sealed record AuthEndpoints(
    EndpointReference Grpc,
    EndpointReference Http
);

public sealed class AuthBuilder(
    IDistributedApplicationBuilder builder
    , IResourceBuilder<PostgresServerResource> postgres
    , string workerClientId
    , string imageTag
    , string proxyImageTag
    , string repoRoot
    , bool useLocal
)
{
    private readonly string? localAppDirectory = GetLocalAppDirectory(repoRoot, useLocal);

    public AuthEndpoints Build(int port, bool isDebug, string grpcReflectionFlag)
    {
        var authDB = postgres.AddDatabase("jobdeck-auth");
        var backend = localAppDirectory is null
            ? ContainerBackend(grpcReflectionFlag, authDB)
            : LocalBackend(isDebug, grpcReflectionFlag, authDB);
        var envoy = EnvoyProxy(backend);
        var proxy = ReverseProxy(port, envoy);
        return new AuthEndpoints(Grpc: proxy, Http: proxy);
    }

    private EndpointReference ReverseProxy(int port, EndpointReference envoy)
    {
        return builder.AddYarp("auth-tls")
            .WithHostHttpsPort(port)
            // Aspire.Hosting.Yarp 13.5.2 does not flatten enum values in cluster configuration.
            .WithEnvironment("REVERSEPROXY__CLUSTERS__cluster_auth-proxy__HTTPREQUEST__VERSIONPOLICY", HttpVersionPolicy.RequestVersionExact.ToString())
            .WithConfiguration(yarp =>
            {
                var grpcCluster = yarp.AddCluster(envoy)
                    .WithForwarderRequestConfig(new()
                    {
                        Version = HttpVersion.Version20,
                        VersionPolicy = HttpVersionPolicy.RequestVersionExact,
                    });
                yarp.AddRoute(grpcCluster);
            })
            .GetEndpoint("https");
    }

    private EndpointReference EnvoyProxy(BackendEndpoints backend)
    {
        return builder.AddContainer(
            name: "auth-proxy",
            image: "ghcr.io/air-hand-org/jobdeck-auth-proxy",
            tag: proxyImageTag
        )
        .WithHttpEndpoint(
            name: "h2c",
            targetPort: 8080
        )
        .WithReference(backend.Grpc)
        .WithReference(backend.Http)
        .WithEnvironment(context =>
        {
            context.EnvironmentVariables["PROXY_PORT"] = "8080";
            context.EnvironmentVariables["AUTH_GRPC_HOST"] = backend.Grpc.Property(EndpointProperty.Host);
            context.EnvironmentVariables["AUTH_GRPC_PORT"] = backend.Grpc.Property(EndpointProperty.TargetPort);
            context.EnvironmentVariables["AUTH_JWKS_HOST"] = backend.Http.Property(EndpointProperty.Host);
            context.EnvironmentVariables["AUTH_JWKS_PORT"] = backend.Http.Property(EndpointProperty.TargetPort);
        })
        .GetEndpoint("h2c");
    }

    private BackendEndpoints ContainerBackend(
        string grpcReflectionFlag,
        IResourceBuilder<PostgresDatabaseResource> authDB
    )
    {
        var authJwtPrivateKey = builder.AddParameter("auth-jwt-private-key", secret: true);
        var authJwtKeyID = builder.AddParameter("auth-jwt-key-id");
        var workerClientSecretHash = builder.AddParameter("job-worker-client-secret-hash", secret: true);
        var testUserSecretHash = builder.AddParameter("test-user-client-secret-hash", secret: true);
        var backend = ConfigureBackend(builder.AddContainer(
            name: "auth",
            image: "ghcr.io/air-hand-org/jobdeck-auth",
            tag: imageTag
        )
        .WithEnvironment("GRPC_PORT", "9091")
        .WithEnvironment("HTTP_PORT", "8081")
        .WithHttpEndpoint(
            name: "grpc-backend",
            targetPort: 9091
        )
        .WithHttpEndpoint(
            name: "http-backend",
            targetPort: 8081
        )
        .AsHttp2Service()
        , authDB,
        grpcReflectionFlag,
        authJwtPrivateKey,
        authJwtKeyID,
        workerClientSecretHash,
        testUserSecretHash
        );
        return new BackendEndpoints(
            Grpc: backend.GetEndpoint("grpc-backend"),
            Http: backend.GetEndpoint("http-backend")
        );
    }

    private BackendEndpoints LocalBackend(
        bool isDebug,
        string grpcReflectionFlag,
        IResourceBuilder<PostgresDatabaseResource> authDB
    )
    {
        var authJwtPrivateKey = builder.AddParameter("auth-jwt-private-key", secret: true);
        var authJwtKeyID = builder.AddParameter("auth-jwt-key-id");
        var workerClientSecretHash = builder.AddParameter("job-worker-client-secret-hash", secret: true);
        var testUserSecretHash = builder.AddParameter("test-user-client-secret-hash", secret: true);
        var migrator = builder.AddGoApp(
            name: "auth-db-migrator",
            appDirectory: localAppDirectory!,
            packagePath: "./cmd/migrator"
        )
        .WithReference(authDB)
        .WithEnvironment("GOWORK", "off")
        .WithEnvironment(context =>
        {
            context.EnvironmentVariables["POSTGRESDB_URI"] = authDB.Resource.UriExpression;
            context.EnvironmentVariables["AUTH_JOB_WORKER_SECRET_HASH"] = workerClientSecretHash;
            context.EnvironmentVariables["AUTH_JOB_WORKER_CLIENT_ID"] = workerClientId;
            context.EnvironmentVariables["AUTH_TEST_USER_SECRET_HASH"] = testUserSecretHash;
        })
        .WaitFor(authDB);

        var backend = ConfigureBackend(builder.AddGoApp(
            name: "auth",
            appDirectory: localAppDirectory!,
            packagePath: "./cmd/server"
        )
        .WithHttpEndpoint(
            name: "grpc-backend",
            env: "GRPC_PORT"
        )
        .WithHttpEndpoint(
            name: "http-backend",
            env: "HTTP_PORT"
        )
        .AsHttp2Service(),
        authDB,
        grpcReflectionFlag,
        authJwtPrivateKey,
        authJwtKeyID,
        workerClientSecretHash,
        testUserSecretHash
        )
        .WithEnvironment("GOWORK", "off")
        .WaitForCompletion(migrator);
        if (isDebug)
        {
            backend.WithDelveServer(new DelveServerOptions
            {
                Port = 2347,
                AcceptMultiClient = true,
                ContinueOnStart = true,
                OnlySameUser = false
            });
        }
        return new BackendEndpoints(
            Grpc: backend.GetEndpoint("grpc-backend"),
            Http: backend.GetEndpoint("http-backend")
        );
    }

    private IResourceBuilder<T> ConfigureBackend<T>(
        IResourceBuilder<T> backend,
        IResourceBuilder<PostgresDatabaseResource> authDB,
        string grpcReflectionFlag,
        IResourceBuilder<ParameterResource> authJwtPrivateKey,
        IResourceBuilder<ParameterResource> authJwtKeyID,
        IResourceBuilder<ParameterResource> workerClientSecretHash,
        IResourceBuilder<ParameterResource> testUserSecretHash
    ) where T : IResource, IResourceWithEnvironment, IResourceWithWaitSupport
    {
        return backend
            .WithReference(authDB)
            .WithEnvironment(context =>
            {
                context.EnvironmentVariables["POSTGRESDB_URI"] = authDB.Resource.UriExpression;
                context.EnvironmentVariables["GRPC_REFLECTION_ENABLED"] = grpcReflectionFlag;
                context.EnvironmentVariables["AUTH_JWT_PRIVATE_KEY"] = authJwtPrivateKey;
                context.EnvironmentVariables["AUTH_JWT_KEY_ID"] = authJwtKeyID;
                context.EnvironmentVariables["AUTH_JOB_WORKER_SECRET_HASH"] = workerClientSecretHash;
                context.EnvironmentVariables["AUTH_JOB_WORKER_CLIENT_ID"] = workerClientId;
                context.EnvironmentVariables["AUTH_TEST_USER_SECRET_HASH"] = testUserSecretHash;
            })
            .WaitFor(authDB);
    }

    private static string? GetLocalAppDirectory(string repoRoot, bool useLocal)
    {
        if (!useLocal)
        {
            return null;
        }
        var path = Path.Combine(repoRoot, "services", "auth");
        if (!Directory.Exists(path))
        {
            return null;
        }
        if (!File.Exists(Path.Combine(path, "go.mod")))
        {
            throw new InvalidOperationException(
                $"The local auth checkout at '{path}' does not contain a go.mod file."
            );
        }
        return path;
    }

    private sealed record BackendEndpoints(EndpointReference Grpc, EndpointReference Http);
}
