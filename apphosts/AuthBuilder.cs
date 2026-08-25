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
)
{
    public EndpointReference Build(int port, bool isDebug, string grpcReflectionFlag)
    {
        var authDB = postgres.AddDatabase("jobdeck-auth");
        var migrator = DBMigration(authDB);
        var backend = Backend(isDebug, grpcReflectionFlag, authDB, migrator);
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
        bool isDebug,
        string grpcReflectionFlag,
        IResourceBuilder<PostgresDatabaseResource> authDB,
        IResourceBuilder<ExecutableResource> dbMigrator
    )
    {
        var authJwtPrivateKey = builder.AddParameter("auth-jwt-private-key", secret: true);
        var backend = builder.AddGoApp(
            name: "auth",
            appDirectory: "../services/auth",
            packagePath: "./cmd/server"
        )
        .WithHttpEndpoint(
            name: "grpc-backend",
            env: "GRPC_PORT"
        )
        .AsHttp2Service()
        .WithReference(authDB)
        .WithEnvironment(context =>
        {
            context.EnvironmentVariables["POSTGRESDB_URI"] = authDB.Resource.UriExpression;
            context.EnvironmentVariables["GRPC_REFLECTION_ENABLED"] = grpcReflectionFlag;
            context.EnvironmentVariables["AUTH_JWT_PRIVATE_KEY"] = authJwtPrivateKey;
        })
        .WaitFor(authDB)
        .WaitForCompletion(dbMigrator)
        ;
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
        return backend.GetEndpoint("grpc-backend");
    }

    private IResourceBuilder<ExecutableResource> DBMigration(IResourceBuilder<PostgresDatabaseResource> authDB)
    {
        var workerClientSecretHash = builder.AddParameter("job-worker-client-secret-hash", secret: true);
        var testUserSecretHash = builder.AddParameter("test-user-client-secret-hash", secret: true);

        return builder.AddGoApp(
            name: "auth-db-migrator",
            appDirectory: "../services/auth",
            packagePath: "./cmd/migrator"
          )
          .WithReference(authDB)
          .WithEnvironment(context =>
          {
              context.EnvironmentVariables["POSTGRESDB_URI"] = authDB.Resource.UriExpression;
              context.EnvironmentVariables["AUTH_JOB_WORKER_SECRET_HASH"] = workerClientSecretHash;
              context.EnvironmentVariables["AUTH_JOB_WORKER_CLIENT_ID"] = workerClientId;
              context.EnvironmentVariables["AUTH_TEST_USER_SECRET_HASH"] = testUserSecretHash;
          })
          .WaitFor(authDB);
    }
}
