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

var postgres = builder.AddPostgres("postgres")
  .WithPgAdmin()
  ;
if (useVolumes)
{
    postgres.WithDataVolume();
}

var IS_DEBUG = string.Equals(builder.Configuration["DEBUG"], "true", StringComparison.OrdinalIgnoreCase);

var enableGrpcReflection = builder.Configuration["GRPC_REFLECTION_ENABLED"] ?? "true";

var apiAuthMode = builder.Configuration["API_AUTH_MODE"] ?? "OPTIONAL"; // fallback

var authGrpcEndpoint = new AuthBuilder(builder, postgres, workerClientId)
    .Build(
        port: 9091,
        isDebug: IS_DEBUG,
        grpcReflectionFlag: enableGrpcReflection
    );

var jobApiGrpcEndpoint = new JobAPIBuilder(
        builder: builder
        , postgres: postgres
    ).Build(
        port: 9090,
        isDebug: IS_DEBUG,
        grpcReflectionFlag: enableGrpcReflection,
        apiAuthMode: apiAuthMode
    )
    ;

var demoContainerEndpoint = new DemoBuilder(builder: builder).Build(8080, KnownNetworkIdentifiers.DefaultAspireContainerNetwork);

new JobWorkerBuilder(
    builder: builder
    , jobApiGrpcEndpoint: jobApiGrpcEndpoint
    , authGrpcEndpoint: authGrpcEndpoint
    , demoContainerEndpoint: demoContainerEndpoint
    , workerClientId: workerClientId
).Build(
    isDebug: IS_DEBUG
);

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
