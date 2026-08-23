using System.Net;
using Microsoft.Extensions.Configuration;
using Aspire.Hosting;
using Aspire.Hosting.ApplicationModel;
using Aspire.Hosting.Yarp;

namespace apphosts;

public sealed class JobAPIBuilder(
    IDistributedApplicationBuilder builder
    , IResourceBuilder<PostgresServerResource> postgres
    , PubSubTopicResource jobWorkerTopic
)
{
    public EndpointReference Build(int port, bool isDebug, string grpcReflectionFlag, string apiAuthMode)
    {
        var db = postgres.AddDatabase("jobdeck");
        var migrator = DBMigration(db);
        var republisher = RepublishOutbox(db, migrator);
        var backend = Backend(isDebug, grpcReflectionFlag, apiAuthMode, db, migrator, republisher, jobWorkerTopic);
        return ReverseProxy(port, backend);
    }

    private EndpointReference ReverseProxy(int port, EndpointReference backend)
    {
        return builder.AddYarp("job-api-tls")
            .WithHostHttpsPort(port)
            // Aspire.Hosting.Yarp 13.4.6 does not flatten enum values in cluster configuration.
            .WithEnvironment("REVERSEPROXY__CLUSTERS__cluster_job-api__HTTPREQUEST__VERSIONPOLICY", HttpVersionPolicy.RequestVersionExact.ToString())
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
        string apiAuthMode,
        IResourceBuilder<PostgresDatabaseResource> db,
        IResourceBuilder<ExecutableResource> dbMigrator,
        IResourceBuilder<ExecutableResource> republisher,
        PubSubTopicResource workerTopic
    )
    {
        // TODO: get pubkey from auth-service
        var authJwtPublicKey = builder.AddParameter("auth-jwt-public-key", secret: false);
        EndpointReference endpoint;
        // TODO: use WithDelveServer after v13.5
        if (isDebug)
        {
            var backend = builder.AddExecutable(
                "job-api",
                "dlv",
                "../services/job-api",
                "debug",
                "./cmd/server",
                "--headless",
                "--api-version=2",
                "--listen=127.0.0.1:2345",
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
              .WithReference(db)
              .WithReference(workerTopic.PubsubEndpoint)
              .WithEnvironment(context =>
              {
                  context.EnvironmentVariables["POSTGRESDB_URI"] = db.Resource.UriExpression;
                  context.EnvironmentVariables["GRPC_REFLECTION_ENABLED"] = grpcReflectionFlag;
                  context.EnvironmentVariables["AUTH_JWT_PUBLIC_KEY"] = authJwtPublicKey;
                  context.EnvironmentVariables["API_AUTH_MODE"] = apiAuthMode;
                  context.EnvironmentVariables["PUBSUB_EMULATOR_HOST"] = workerTopic.PubsubEndpoint.Property(EndpointProperty.HostAndPort);
                  context.EnvironmentVariables["PUBSUB_PROJECT_ID"] = workerTopic.ProjectID;
                  context.EnvironmentVariables["PUBSUB_WORKER_TOPIC_ID"] = workerTopic.TopicID;
              })
              .WaitFor(db)
              .WaitForCompletion(dbMigrator)
              .WaitForCompletion(republisher)
              .WaitForCompletion(workerTopic.Creator)
              ;

            endpoint = backend.GetEndpoint("grpc-backend");
        }
        else
        {
            var backend = builder.AddGoApp(
                name: "job-api",
                appDirectory: "../services/job-api",
                packagePath: "./cmd/server"
              )
              //    .WithDelveServer(port: 2345) // --only-same-user=false needed
              .WithHttpEndpoint(
                  name: "grpc-backend",
                  env: "GRPC_PORT"
              )
              .AsHttp2Service()
              .WithReference(db)
              .WithReference(workerTopic.PubsubEndpoint)
              .WithEnvironment(context =>
              {
                  context.EnvironmentVariables["POSTGRESDB_URI"] = db.Resource.UriExpression;
                  context.EnvironmentVariables["GRPC_REFLECTION_ENABLED"] = grpcReflectionFlag;
                  context.EnvironmentVariables["AUTH_JWT_PUBLIC_KEY"] = authJwtPublicKey;
                  context.EnvironmentVariables["API_AUTH_MODE"] = apiAuthMode;
                  context.EnvironmentVariables["PUBSUB_EMULATOR_HOST"] = workerTopic.PubsubEndpoint.Property(EndpointProperty.HostAndPort);
                  context.EnvironmentVariables["PUBSUB_PROJECT_ID"] = workerTopic.ProjectID;
                  context.EnvironmentVariables["PUBSUB_WORKER_TOPIC_ID"] = workerTopic.TopicID;
              })
              .WaitFor(db)
              .WaitForCompletion(dbMigrator)
              .WaitForCompletion(republisher)
              .WaitForCompletion(workerTopic.Creator)
              ;

            endpoint = backend.GetEndpoint("grpc-backend");
        }
        return endpoint;
    }

    private IResourceBuilder<ExecutableResource> DBMigration(IResourceBuilder<PostgresDatabaseResource> db)
    {
        return builder.AddGoApp(
            name: "job-db-migrator",
            appDirectory: "../services/job-api",
            packagePath: "./cmd/migrator"
          )
          .WithReference(db)
          .WithEnvironment(context =>
          {
              context.EnvironmentVariables["POSTGRESDB_URI"] = db.Resource.UriExpression;
          })
          .WaitFor(db);
    }

    private IResourceBuilder<ExecutableResource> RepublishOutbox(IResourceBuilder<PostgresDatabaseResource> db, IResourceBuilder<ExecutableResource> migrator)
    {
        return builder.AddGoApp(
                name: "job-outbox-republisher",
                appDirectory: "../services/job-api",
                packagePath: "./cmd/republish"
            )
            .WithReference(db)
            .WithEnvironment(context =>
            {
                context.EnvironmentVariables["POSTGRESDB_URI"] = db.Resource.UriExpression;
            })
            .WaitForCompletion(migrator)
        ;
    }
}
