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
        var proxy = ReverseProxy(port, backend.Endpoint);
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
.WithEnvironment("PORT", "8080")
        .WithHttpEndpoint(
            name: "backend",
            targetPort: 8080
        )
        .AsHttp2Service()
        , authDB,
        grpcReflectionFlag,
        authJwtPrivateKey,
        authJwtKeyID,
        workerClientSecretHash,
        testUserSecretHash
        );
        return new BackendEndpoints(backend.GetEndpoint("backend"));
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
            name: "backend",
            env: "PORT"
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
        return new BackendEndpoints(backend.GetEndpoint("backend"));
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

    private sealed record BackendEndpoints(EndpointReference Endpoint);
}
