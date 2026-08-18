using System.Net;
using Microsoft.Extensions.Configuration;
using Aspire.Hosting;
using Aspire.Hosting.ApplicationModel;
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
            // Aspire.Hosting.Yarp 13.4.6 does not flatten enum values in cluster configuration.
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
        EndpointReference endpoint;
        // TODO: use WithDelveServer after v13.5
        if (isDebug)
        {
            var backend = builder.AddExecutable(
                "auth",
                "dlv",
                "../services/auth",
                "debug",
                "./cmd/server",
                "--headless",
                "--api-version=2",
                "--listen=127.0.0.1:2347",
                "--accept-multiclient",
                "--log",
                "--log-output=rpc,dap,debugger",
                "--only-same-user=false" // need by Zed on WSL2?
                , "--continue"
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

            endpoint = backend.GetEndpoint("grpc-backend");
        }
        else
        {
            var backend = builder.AddGoApp(
                name: "auth",
                appDirectory: "../services/auth",
                packagePath: "./cmd/server"
              )
              //    .WithDelveServer(port: 2345) // --only-same-user=false needed
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

            endpoint = backend.GetEndpoint("grpc-backend");
        }
        return endpoint;
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
