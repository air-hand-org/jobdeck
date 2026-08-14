using Microsoft.Extensions.Configuration;
using Aspire.Hosting.ApplicationModel;

var builder = DistributedApplication.CreateBuilder(args);

var repoRoot = Path.GetFullPath(
    Path.Combine(builder.Environment.ContentRootPath, "..")
);

var authJwtPrivateKey = builder.AddParameter("auth-jwt-private-key", secret: true);
var authJwtPublicKey = builder.AddParameter("auth-jwt-public-key", secret: false);
var workerClientSecret = builder.AddParameter("job-worker-client-secret", secret: true);
var workerClientSecretHash = builder.AddParameter("job-worker-client-secret-hash", secret: true);
var workerClientId = "job-worker";
var testUserSecretHash = builder.AddParameter("test-user-client-secret-hash", secret: true);

var useVolumes = builder.Configuration.GetValue<bool>("UseVolumes", true);

var compose = builder.AddDockerComposeEnvironment("compose")
  .WithProperties(env =>
  {
      env.DashboardEnabled = true;
  })
  ;

var postgres = builder.AddPostgres("postgres")
  .WithPgAdmin()
  .PublishAsDockerComposeService((resource, service) =>
  {
      service.Name = "db";
  })
  ;
if (useVolumes)
{
    postgres.WithDataVolume();
}

var job_db = postgres.AddDatabase("jobdeck");
var job_db_migration = builder.AddGoApp(
    name: "job-db-migrator",
    appDirectory: "../services/job-api",
    packagePath: "./cmd/migrator"
  )
  .WithReference(job_db)
  .WithEnvironment(context =>
  {
      context.EnvironmentVariables["POSTGRESDB_URI"] = job_db.Resource.UriExpression;
  })
  .WaitFor(job_db)
  .PublishAsDockerComposeService((resource, service) =>
  {
      service.Name = "job-db-migrator";
  })
  ;

var auth_db = postgres.AddDatabase("jobdeck-auth");
var auth_db_migration = builder.AddGoApp(
    name: "auth-db-migrator",
    appDirectory: "../services/auth",
    packagePath: "./cmd/migrator"
  )
  .WithReference(auth_db)
  .WithEnvironment(context =>
  {
      context.EnvironmentVariables["POSTGRESDB_URI"] = auth_db.Resource.UriExpression;
      context.EnvironmentVariables["AUTH_JOB_WORKER_SECRET_HASH"] = workerClientSecretHash;
      context.EnvironmentVariables["AUTH_JOB_WORKER_CLIENT_ID"] = workerClientId;
      context.EnvironmentVariables["AUTH_TEST_USER_SECRET_HASH"] = testUserSecretHash;
  })
  .WaitFor(auth_db)
  .PublishAsDockerComposeService((resource, service) =>
  {
      service.Name = "auth-db-migrator";
  })
  ;

EndpointReference jobApiGrpcEndpoint;
EndpointReference authGrpcEndpoint;

var IS_DEBUG = string.Equals(builder.Configuration["DEBUG"], "true", StringComparison.OrdinalIgnoreCase);

var enableGrpcReflection = builder.Configuration["GRPC_REFLECTION_ENABLED"] ?? "true";

var apiAuthMode = builder.Configuration["API_AUTH_MODE"] ?? "OPTIONAL"; // fallback

// TODO: use WithDelveServer after v13.5
if (IS_DEBUG)
{
    var auth = builder.AddExecutable(
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
      .WithHttpsEndpoint(
          port: 9091,
          name: "grpc",
          env: "GRPC_PORT",
          isProxied: true
      )
      .WithReference(auth_db)
      .WithEnvironment(context =>
      {
          context.EnvironmentVariables["POSTGRESDB_URI"] = auth_db.Resource.UriExpression;
          context.EnvironmentVariables["GRPC_REFLECTION_ENABLED"] = enableGrpcReflection;
          context.EnvironmentVariables["AUTH_JWT_PRIVATE_KEY"] = authJwtPrivateKey;
      })
      .WaitFor(auth_db)
      .WaitForCompletion(auth_db_migration)
      .PublishAsDockerComposeService((resource, service) =>
      {
          service.Name = "auth";
      });

    authGrpcEndpoint = auth.GetEndpoint("grpc");
}
else
{
    var auth = builder.AddGoApp(
        name: "auth",
        appDirectory: "../services/auth",
        packagePath: "./cmd/server"
      )
      //    .WithDelveServer(port: 2345) // --only-same-user=false needed
      .WithHttpsEndpoint(
          port: 9091,
          name: "grpc",
          env: "GRPC_PORT",
          isProxied: true
      )
      .WithReference(auth_db)
      .WithEnvironment(context =>
      {
          context.EnvironmentVariables["POSTGRESDB_URI"] = auth_db.Resource.UriExpression;
          context.EnvironmentVariables["GRPC_REFLECTION_ENABLED"] = enableGrpcReflection;
          context.EnvironmentVariables["AUTH_JWT_PRIVATE_KEY"] = authJwtPrivateKey;
      })
      .WaitFor(auth_db)
      .WaitForCompletion(auth_db_migration)
      .PublishAsDockerComposeService((resource, service) =>
      {
          service.Name = "auth";
      });

    authGrpcEndpoint = auth.GetEndpoint("grpc");
}

if (IS_DEBUG)
{
    var job_api = builder.AddExecutable(
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
      .WithHttpsEndpoint(
          port: 9090,
          name: "grpc",
          env: "GRPC_PORT",
          isProxied: true
      )
      .WithReference(job_db)
      .WithEnvironment(context =>
      {
          context.EnvironmentVariables["POSTGRESDB_URI"] = job_db.Resource.UriExpression;
          context.EnvironmentVariables["GRPC_REFLECTION_ENABLED"] = enableGrpcReflection;
          context.EnvironmentVariables["AUTH_JWT_PUBLIC_KEY"] = authJwtPublicKey;
          context.EnvironmentVariables["API_AUTH_MODE"] = apiAuthMode;
      })
      .WaitFor(job_db)
      .WaitForCompletion(job_db_migration)
      .PublishAsDockerComposeService((resource, service) =>
      {
          service.Name = "job-api";
      });

    jobApiGrpcEndpoint = job_api.GetEndpoint("grpc");
}
else
{
    var job_api = builder.AddGoApp(
        name: "job-api",
        appDirectory: "../services/job-api",
        packagePath: "./cmd/server"
      )
      //    .WithDelveServer(port: 2345) // --only-same-user=false needed
      .WithHttpsEndpoint(
          port: 9090,
          name: "grpc",
          env: "GRPC_PORT",
          isProxied: true
      )
      .WithReference(job_db)
      .WithEnvironment(context =>
      {
          context.EnvironmentVariables["POSTGRESDB_URI"] = job_db.Resource.UriExpression;
          context.EnvironmentVariables["GRPC_REFLECTION_ENABLED"] = enableGrpcReflection;
          context.EnvironmentVariables["AUTH_JWT_PUBLIC_KEY"] = authJwtPublicKey;
          context.EnvironmentVariables["API_AUTH_MODE"] = apiAuthMode;
      })
      .WaitFor(job_db)
      .WaitForCompletion(job_db_migration)
      .PublishAsDockerComposeService((resource, service) =>
      {
          service.Name = "job-api";
      });

    jobApiGrpcEndpoint = job_api.GetEndpoint("grpc");
}

var isGitHubActions = string.Equals(
    Environment.GetEnvironmentVariable("GITHUB_ACTIONS"),
    "true",
    StringComparison.OrdinalIgnoreCase
);

var demo = builder.AddGoApp(
    name: "demo",
    appDirectory: "../services/demo",
    packagePath: "./cmd/server"
  )
  .WithExternalHttpEndpoints()
  .WithHttpsEndpoint(
      name: "https",
      port: 8080,
      env: "PORT",
      isProxied: true
  )
  .PublishAsDockerComposeService((resource, service) =>
  {
      service.Name = "demo";
  });

var demoEndpoint = demo.GetEndpoint("https", KnownNetworkIdentifiers.LocalhostNetwork);

var forwardProxy = builder.AddDockerfile(
        name: "forward-proxy",
        contextPath: "../containers/forward-proxy"
    )
    .WithHttpEndpoint(
        name: "proxy",
        port: 3128,
        targetPort: 3128
    )
    .WithEnvironment(context =>
    {
        context.EnvironmentVariables["SQUID_ALLOWED_DEMO_HOST_AND_PORT"] = ReferenceExpression.Create($"host.docker.internal:{demoEndpoint.Property(EndpointProperty.Port)}");
    })
    .PublishAsDockerComposeService((resource, service) =>
    {
        service.Name = "forward-proxy";
    })
;
if (isGitHubActions)
{
    forwardProxy.WithContainerRuntimeArgs(
        "--add-host=host.docker.internal:host-gateway"
    );
}
var forwardProxyEndpoint = forwardProxy.GetEndpoint("proxy");

if (IS_DEBUG)
{
    var job_worker = builder.AddExecutable(
        "job-worker",
        "dlv",
        "../services/job-worker",
        "debug",
        "./cmd/worker",
        "--headless",
        "--api-version=2",
        "--listen=127.0.0.1:2346",
        "--accept-multiclient",
        "--log",
        "--log-output=rpc,dap,debugger",
        "--only-same-user=false" // need by Zed on WSL2?
        , "--continue"
        , "--"
        , "-worker", "3"
      )
      .WithReference(jobApiGrpcEndpoint)
      .WithReference(authGrpcEndpoint)
      .WithReference(forwardProxyEndpoint)
      .WithEnvironment(context =>
      {
          context.EnvironmentVariables["JOB_API_GRPC_ENDPOINT"] = jobApiGrpcEndpoint.Property(EndpointProperty.Url);
          context.EnvironmentVariables["AUTH_GRPC_ENDPOINT"] = authGrpcEndpoint.Property(EndpointProperty.Url);
          context.EnvironmentVariables["AUTH_CLIENT_ID"] = workerClientId;
          context.EnvironmentVariables["AUTH_CLIENT_SECRET"] = workerClientSecret;
          context.EnvironmentVariables["WORKER_HTTP_PROXY"] = forwardProxyEndpoint.Property(EndpointProperty.Url);
      })
      // .WaitFor(job_api)
      .PublishAsDockerComposeService((resource, service) =>
      {
          service.Name = "job-worker";
      });
}
else
{
    var job_worker = builder.AddGoApp(
        name: "job-worker",
        appDirectory: "../services/job-worker",
        packagePath: "./cmd/worker"
      )
      .WithAppArgs("-worker", "3")
      //.WithReference(job_api) // WARN: ExecutableReference is not supported.
      .WithReference(jobApiGrpcEndpoint)
      .WithReference(authGrpcEndpoint)
      .WithReference(forwardProxyEndpoint)
      .WithEnvironment(context =>
      {
          context.EnvironmentVariables["JOB_API_GRPC_ENDPOINT"] = jobApiGrpcEndpoint.Property(EndpointProperty.Url);
          context.EnvironmentVariables["AUTH_GRPC_ENDPOINT"] = authGrpcEndpoint.Property(EndpointProperty.Url);
          context.EnvironmentVariables["AUTH_CLIENT_ID"] = "job-worker";
          context.EnvironmentVariables["AUTH_CLIENT_SECRET"] = workerClientSecret;
          context.EnvironmentVariables["WORKER_HTTP_PROXY"] = forwardProxyEndpoint.Property(EndpointProperty.Url);
      })
      //.WaitFor(job_api)
      .PublishAsDockerComposeService((resource, service) =>
      {
          service.Name = "job-worker";
      });
}

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
  // .WaitFor(job_api)
  .PublishAsDockerComposeService((resource, service) =>
  {
      service.Name = "ui";
  })
;

builder.Build().Run();
