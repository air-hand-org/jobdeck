using System.Net;
using Microsoft.Extensions.Configuration;
using Aspire.Hosting;
using Aspire.Hosting.ApplicationModel;
using Aspire.Hosting.Go;
using Aspire.Hosting.Yarp;

namespace apphosts;

public sealed class JobAPIBuilder(
    IDistributedApplicationBuilder builder
    , IResourceBuilder<PostgresServerResource> postgres
    , PubSubTopicResource jobWorkerTopic
    , EndpointReference authHttpEndpoint
    , string imageTag
    , string repoRoot
    , bool useLocal
)
{
    private readonly string? localAppDirectory = GetLocalAppDirectory(repoRoot, useLocal);

    public EndpointReference Build(int port, bool isDebug, string grpcReflectionFlag, string apiAuthMode)
    {
        var db = postgres.AddDatabase("jobdeck");
        var backend = localAppDirectory is null
            ? ContainerBackend(grpcReflectionFlag, apiAuthMode, db, jobWorkerTopic)
            : LocalBackend(isDebug, grpcReflectionFlag, apiAuthMode, db, jobWorkerTopic);
        return ReverseProxy(port, backend);
    }

    private EndpointReference ReverseProxy(int port, EndpointReference backend)
    {
        return builder.AddYarp("job-api-tls")
            .WithHostHttpsPort(port)
            // Aspire.Hosting.Yarp 13.5.2 does not flatten enum values in cluster configuration.
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

    private EndpointReference ContainerBackend(
        string grpcReflectionFlag,
        string apiAuthMode,
        IResourceBuilder<PostgresDatabaseResource> db,
        PubSubTopicResource workerTopic
    )
    {
        return ConfigureBackend(builder.AddContainer(
            name: "job-api",
            image: "ghcr.io/air-hand-org/jobdeck-job-api",
            tag: imageTag
        )
        .WithHttpEndpoint(
            name: "grpc-backend",
            targetPort: 9090
        )
        .AsHttp2Service(),
        db,
        workerTopic,
        authHttpEndpoint,
        grpcReflectionFlag,
        apiAuthMode
        ).GetEndpoint("grpc-backend");
    }

    private EndpointReference LocalBackend(
        bool isDebug,
        string grpcReflectionFlag,
        string apiAuthMode,
        IResourceBuilder<PostgresDatabaseResource> db,
        PubSubTopicResource workerTopic
    )
    {
        var migrator = DBMigration(db);
        var republisher = RepublishOutbox(db, migrator);
        var backend = ConfigureBackend(builder.AddGoApp(
            name: "job-api",
            appDirectory: localAppDirectory!,
            packagePath: "./cmd/server"
        )
        .WithHttpEndpoint(
            name: "grpc-backend",
            env: "GRPC_PORT"
        )
        .AsHttp2Service(),
        db,
        workerTopic,
        authHttpEndpoint,
        grpcReflectionFlag,
        apiAuthMode
        )
        .WithEnvironment("GOWORK", "off")
        .WaitForCompletion(migrator)
        .WaitForCompletion(republisher);

        if (isDebug)
        {
            backend.WithDelveServer(new DelveServerOptions
            {
                Port = 2345,
                AcceptMultiClient = true,
                ContinueOnStart = true,
                OnlySameUser = false
            });
        }

        return backend.GetEndpoint("grpc-backend");
    }

    private IResourceBuilder<ExecutableResource> DBMigration(IResourceBuilder<PostgresDatabaseResource> db)
    {
        return builder.AddGoApp(
            name: "job-db-migrator",
            appDirectory: localAppDirectory!,
            packagePath: "./cmd/migrator"
          )
          .WithReference(db)
          .WithEnvironment("GOWORK", "off")
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
                appDirectory: localAppDirectory!,
                packagePath: "./cmd/republish"
            )
            .WithReference(db)
            .WithEnvironment("GOWORK", "off")
            .WithEnvironment(context =>
            {
                context.EnvironmentVariables["POSTGRESDB_URI"] = db.Resource.UriExpression;
            })
            .WaitForCompletion(migrator)
        ;
    }

    private IResourceBuilder<T> ConfigureBackend<T>(
        IResourceBuilder<T> backend,
        IResourceBuilder<PostgresDatabaseResource> db,
        PubSubTopicResource workerTopic,
        EndpointReference authHttpEndpoint,
        string grpcReflectionFlag,
        string apiAuthMode
    ) where T : IResource, IResourceWithEnvironment, IResourceWithWaitSupport
    {
        return backend
            .WithReference(db)
            .WithReference(workerTopic.PubsubEndpoint)
            .WithReference(authHttpEndpoint)
            .WithEnvironment(context =>
            {
                context.EnvironmentVariables["POSTGRESDB_URI"] = db.Resource.UriExpression;
                context.EnvironmentVariables["GRPC_REFLECTION_ENABLED"] = grpcReflectionFlag;
                context.EnvironmentVariables["AUTH_HTTP_ENDPOINT"] = authHttpEndpoint.Property(EndpointProperty.Url);
                context.EnvironmentVariables["API_AUTH_MODE"] = apiAuthMode;
                context.EnvironmentVariables["PUBSUB_EMULATOR_HOST"] = workerTopic.PubsubEndpoint.Property(EndpointProperty.HostAndPort);
                context.EnvironmentVariables["PUBSUB_PROJECT_ID"] = workerTopic.ProjectID;
                context.EnvironmentVariables["PUBSUB_WORKER_TOPIC_ID"] = workerTopic.TopicID;
            })
            .WaitFor(db)
            .WaitForCompletion(workerTopic.Creator);
    }

    private static string? GetLocalAppDirectory(string repoRoot, bool useLocal)
    {
        if (!useLocal)
        {
            return null;
        }
        var path = Path.Combine(repoRoot, "services", "job-api");
        if (!Directory.Exists(path))
        {
            return null;
        }
        if (!File.Exists(Path.Combine(path, "go.mod")))
        {
            throw new InvalidOperationException(
                $"The local job-api checkout at '{path}' does not contain a go.mod file."
            );
        }
        return path;
    }
}
