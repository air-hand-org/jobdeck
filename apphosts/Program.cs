using System.Net;
using Microsoft.Extensions.Configuration;
using Aspire.Hosting.ApplicationModel;
using Aspire.Hosting.Yarp;
using YamlDotNet.Serialization;
using YamlDotNet.Serialization.NamingConventions;

using apphosts;

var builder = DistributedApplication.CreateBuilder(args);

var repoRoot = Path.GetFullPath(
    Path.Combine(builder.Environment.ContentRootPath, "..")
);

var production = new DeserializerBuilder()
    .WithNamingConvention(CamelCaseNamingConvention.Instance)
    .IgnoreUnmatchedProperties()
    .Build()
    .Deserialize<ProductionManifest>(File.ReadAllText(
        Path.Combine(repoRoot, "environments", "production.yaml")
    ));
var authImageTag = $"v{production.Images["auth"].Version}";
var authProxyImageTag = $"v{production.Images["auth-proxy"].Version}";
var uiImageTag = $"v{production.Images["ui"].Version}";
var demoImageTag = $"v{production.Images["demo"].Version}";
var jobApiImageTag = $"v{production.Images["job-api"].Version}";
var jobWorkerImageTag = $"v{production.Images["job-worker"].Version}";
var forwardProxyImageTag = $"v{production.Images["forward-proxy"].Version}";

var workerClientId = "job-worker";

var useVolumes = builder.Configuration.GetValue<bool>("UseVolumes", true);

var useLocal = builder.Configuration.GetValue<bool>("USE_LOCAL");

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

var authEndpoints = new AuthBuilder(
    builder,
    postgres,
    workerClientId,
    authImageTag,
    authProxyImageTag,
    repoRoot,
    useLocal
)
    .Build(
        port: 9091,
        isDebug: IS_DEBUG,
        grpcReflectionFlag: enableGrpcReflection
    );
var authGrpcEndpoint = authEndpoints.Grpc;

var jobApiGrpcEndpoint = new JobAPIBuilder(
        builder: builder,
        postgres: postgres,
        jobWorkerTopic: pubsubWorkerTopic.Topic,
        authHttpEndpoint: authEndpoints.Http,
        imageTag: jobApiImageTag,
        repoRoot: repoRoot,
        useLocal: useLocal
    ).Build(
        port: 9090,
        isDebug: IS_DEBUG,
        grpcReflectionFlag: enableGrpcReflection,
        apiAuthMode: apiAuthMode
    )
    ;

var demoContainerEndpoint = new DemoBuilder(
    builder: builder,
    imageTag: demoImageTag,
    repoRoot: repoRoot,
    useLocal: useLocal
).Build(8080, KnownNetworkIdentifiers.DefaultAspireContainerNetwork);

var workerPubsubVerificationToken = Guid.NewGuid().ToString("N");

var workerPushEndpoint = new JobWorkerBuilder(
    builder: builder
    , jobApiGrpcEndpoint: jobApiGrpcEndpoint
    , authGrpcEndpoint: authGrpcEndpoint
    , demoContainerEndpoint: demoContainerEndpoint
    , workerClientId: workerClientId
    , pubsubVerificationToken: workerPubsubVerificationToken
    , imageTag: jobWorkerImageTag
    , forwardProxyImageTag: forwardProxyImageTag
    , repoRoot: repoRoot
    , useLocal: useLocal
).Build(
    port: 9092,
    isDebug: IS_DEBUG
);

pubsubWorkerTopic.PushSubscription("job-worker-subscription", workerPushEndpoint, workerPubsubVerificationToken);

new UiBuilder(
    builder: builder,
    jobApiGrpcEndpoint: jobApiGrpcEndpoint,
    authGrpcEndpoint: authGrpcEndpoint,
    imageTag: uiImageTag,
    repoRoot: repoRoot,
    useLocal: useLocal
).Build();

builder.Build().Run();

file sealed class ProductionManifest
{
    public required Dictionary<string, ProductionImage> Images { get; init; }
}

file sealed class ProductionImage
{
    public required string Version { get; init; }
}
