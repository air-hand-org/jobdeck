using Aspire.Hosting.ApplicationModel;

var builder = DistributedApplication.CreateBuilder(args);

var compose = builder.AddDockerComposeEnvironment("compose")
  .WithProperties(env => {
    env.DashboardEnabled = true;
  })
  ;

var postgres = builder.AddPostgres("postgres")
  .WithDataVolume()
  .WithPgAdmin()
  .PublishAsDockerComposeService((resource, service) =>
  {
    service.Name = "db";
  })
  ;
var db = postgres.AddDatabase("jobdeck");
var migration = builder.AddGoApp(
    name: "job-db-migrator",
    appDirectory: "../services/job-api",
    packagePath: "./cmd/migrator"
  )
  .WithReference(db)
  .WithEnvironment(context => {
    context.EnvironmentVariables["POSTGRESDB_URI"] = db.Resource.UriExpression;
  })
  .WaitFor(db)
  .PublishAsDockerComposeService((resource, service) => {
    service.Name = "migration";
  })
  ;


EndpointReference jobApiGrpcEndpoint;

var IS_DEBUG = string.Equals(builder.Configuration["DEBUG"], "true", StringComparison.OrdinalIgnoreCase);

// TODO: use WithDelveServer after v13.5
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
    .WithHttpEndpoint(
        port: 9090,
        targetPort: 9090,
        name: "grpc",
        env: "GRPC_PORT",
        isProxied: false
    )
    .WithReference(db)
    .WithEnvironment(context =>
    {
      context.EnvironmentVariables["POSTGRESDB_URI"] = db.Resource.UriExpression;
    })
    .WaitFor(db)
    .WaitForCompletion(migration)
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
    .WithHttpEndpoint(
        port: 9090,
        targetPort: 9090,
        name: "grpc",
        env: "GRPC_PORT",
        isProxied: false
    )
    .WithReference(db)
    .WithEnvironment(context =>
    {
      context.EnvironmentVariables["POSTGRESDB_URI"] = db.Resource.UriExpression;
    })
    .WaitFor(db)
    .WaitForCompletion(migration)
    .PublishAsDockerComposeService((resource, service) =>
    {
      service.Name = "job-api";
    });

  jobApiGrpcEndpoint = job_api.GetEndpoint("grpc");
}

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
    .WithEnvironment(context => {
      context.EnvironmentVariables["JOB_API_SERVER_URI"] = jobApiGrpcEndpoint.Property(EndpointProperty.HostAndPort);
    })
    // .WaitFor(job_api)
    .PublishAsDockerComposeService((resource, service) => {
      service.Name = "job-worker";
    });
} else {
  var job_worker = builder.AddGoApp(
      name: "job-worker",
      appDirectory: "../services/job-worker",
      packagePath: "./cmd/worker"
    )
    .WithAppArgs("-worker", "3")
    //.WithReference(job_api) // WARN: ExecutableReference is not supported.
    .WithReference(jobApiGrpcEndpoint)
    .WithEnvironment(context => {
      context.EnvironmentVariables["JOB_API_SERVER_URI"] = jobApiGrpcEndpoint.Property(EndpointProperty.HostAndPort);
    })
    //.WaitFor(job_api)
    .PublishAsDockerComposeService((resource, service) => {
      service.Name = "job-worker";
    });
}

var ui = builder.AddProject("ui", "../services/ui/ui.csproj")
  .WithExternalHttpEndpoints()
  .WithReference(jobApiGrpcEndpoint)
  .WithEnvironment(context =>
  {
    context.EnvironmentVariables["JOB_API_GRPC_ENDPOINT"] = jobApiGrpcEndpoint.Property(EndpointProperty.Url);
  })
  // .WaitFor(job_api)
  .PublishAsDockerComposeService((resource, service) =>
  {
    service.Name = "ui";
  })
;

builder.Build().Run();
