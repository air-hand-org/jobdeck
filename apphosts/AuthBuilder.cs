using System.Net;
using Microsoft.Extensions.Configuration;
using Aspire.Hosting;
using Aspire.Hosting.ApplicationModel;
using Aspire.Hosting.Go;
using Aspire.Hosting.Yarp;

namespace apphosts;

public sealed class AuthBuilder(
    IDistributedApplicationBuilder builder
    , IResourceBuilder<PostgresServerResource> postgres
    , string workerClientId
    , string imageTag
)
{
    public EndpointReference Build(int port, string grpcReflectionFlag)
    {
        var authDB = postgres.AddDatabase("jobdeck-auth");
        var backend = Backend(grpcReflectionFlag, authDB);
        return ReverseProxy(port, backend);
    }

    private EndpointReference ReverseProxy(int port, EndpointReference backend)
    {
        return builder.AddYarp("auth-tls")
            .WithHostHttpsPort(port)
            // Aspire.Hosting.Yarp 13.5.2 does not flatten enum values in cluster configuration.
            .WithEnvironment("REVERSEPROXY__CLUSTERS__cluster_auth__HTTPREQUEST__VERSIONPOLICY", HttpVersionPolicy.RequestVersionExact.ToString())
            .WithConfiguration(yarp =>
            {
                var cluster = yarp.AddCluster(backend)
                    .WithForwarderRequestConfig(new()
                    {
                        Version = HttpVersion.Version20,
                        VersionPolicy = HttpVersionPolicy.RequestVersionExact,
                    });
                yarp.AddRoute(cluster);
            })
            .GetEndpoint("https");
    }

    private EndpointReference Backend(
        string grpcReflectionFlag,
        IResourceBuilder<PostgresDatabaseResource> authDB
    )
    {
        var authJwtPrivateKey = builder.AddParameter("auth-jwt-private-key", secret: true);
        var workerClientSecretHash = builder.AddParameter("job-worker-client-secret-hash", secret: true);
        var testUserSecretHash = builder.AddParameter("test-user-client-secret-hash", secret: true);
        var backend = builder.AddContainer(
            name: "auth",
            image: "ghcr.io/air-hand-org/jobdeck-auth",
            tag: imageTag
        )
        .WithHttpEndpoint(
            name: "grpc-backend",
            targetPort: 9091
        )
        .AsHttp2Service()
        .WithReference(authDB)
        .WithEnvironment(context =>
        {
            context.EnvironmentVariables["POSTGRESDB_URI"] = authDB.Resource.UriExpression;
            context.EnvironmentVariables["GRPC_REFLECTION_ENABLED"] = grpcReflectionFlag;
            context.EnvironmentVariables["AUTH_JWT_PRIVATE_KEY"] = authJwtPrivateKey;
            context.EnvironmentVariables["AUTH_JOB_WORKER_SECRET_HASH"] = workerClientSecretHash;
            context.EnvironmentVariables["AUTH_JOB_WORKER_CLIENT_ID"] = workerClientId;
            context.EnvironmentVariables["AUTH_TEST_USER_SECRET_HASH"] = testUserSecretHash;
        })
        .WaitFor(authDB)
        ;
        return backend.GetEndpoint("grpc-backend");
    }
}
