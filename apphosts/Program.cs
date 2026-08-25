using System.Net;
using Microsoft.Extensions.Configuration;
using Aspire.Hosting.ApplicationModel;
using Aspire.Hosting.Yarp;

using apphosts;

var builder = DistributedApplication.CreateBuilder(args);

var repoRoot = Path.GetFullPath(
    Path.Combine(builder.Environment.ContentRootPath, "..")
);

var workerClientId = "job-worker";

var useVolumes = builder.Configuration.GetValue<bool>("UseVolumes", true);

var IS_DEBUG = string.Equals(builder.Configuration["DEBUG"], "true", StringComparison.OrdinalIgnoreCase);

var enableGrpcReflection = builder.Configuration["GRPC_REFLECTION_ENABLED"] ?? "true";

var apiAuthMode = builder.Configuration["API_AUTH_MODE"] ?? "OPTIONAL"; // fallback

var postgres = builder.AddPostgres("postgres")
  .WithImageTag("18.4-bookworm")
  .WithPgAdmin()
  ;
if (useVolumes)
{
    postgres.WithDataVolume();
}

var pubsubBuilder = new PubSubBuilder(builder);

var pubsub = pubsubBuilder.PubSub();
var pubsubWorkerTopic = pubsub.Topic("job-worker-topic");

var authGrpcEndpoint = new AuthBuilder(builder, postgres, workerClientId)
    .Build(
        port: 9091,
        isDebug: IS_DEBUG,
        grpcReflectionFlag: enableGrpcReflection
    );

var jobApiGrpcEndpoint = new JobAPIBuilder(
        builder: builder,
        postgres: postgres,
        jobWorkerTopic: pubsubWorkerTopic.Topic
    ).Build(
        port: 9090,
        isDebug: IS_DEBUG,
        grpcReflectionFlag: enableGrpcReflection,
        apiAuthMode: apiAuthMode
    )
    ;

var demoContainerEndpoint = new DemoBuilder(builder: builder).Build(8080, KnownNetworkIdentifiers.DefaultAspireContainerNetwork);

var workerPubsubVerificationToken = Guid.NewGuid().ToString("N");

var workerPushEndpoint = new JobWorkerBuilder(
    builder: builder
    , jobApiGrpcEndpoint: jobApiGrpcEndpoint
    , authGrpcEndpoint: authGrpcEndpoint
    , demoContainerEndpoint: demoContainerEndpoint
    , workerClientId: workerClientId
    , pubsubVerificationToken: workerPubsubVerificationToken
).Build(
    port: 9092,
    isDebug: IS_DEBUG
);

pubsubWorkerTopic.PushSubscription("job-worker-subscription", workerPushEndpoint, workerPubsubVerificationToken);

var ui = builder.AddProject("ui", "../services/ui/ui.csproj")
  .WithExternalHttpEndpoints()
  .WithReference(jobApiGrpcEndpoint)
  .WithReference(authGrpcEndpoint)
  .WithEnvironment(context =>
  {
      context.EnvironmentVariables["JOB_API_GRPC_ENDPOINT"] = jobApiGrpcEndpoint.Property(EndpointProperty.Url);
      context.EnvironmentVariables["AUTH_GRPC_ENDPOINT"] = authGrpcEndpoint.Property(EndpointProperty.Url);
  })
  .WithHttpHealthCheck("/health/ready")
;

builder.Build().Run();
