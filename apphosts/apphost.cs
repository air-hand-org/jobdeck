#:package Aspire.Hosting.Docker@13.4.4
#:package Aspire.Hosting.Go@13.4.4-preview.1.26314.3
#:package Aspire.Hosting.Kubernetes@13.4.4-preview.1.26314.3
#:package Aspire.Hosting.PostgreSQL@13.4.4
#:sdk Aspire.AppHost.Sdk@13.4.4+4f218933552e18ff2874d1b6d5dc3fe671e3b6d9

var builder = DistributedApplication.CreateBuilder(args);

var compose = builder.AddDockerComposeEnvironment("compose")
  .WithProperties(env => {
    env.DashboardEnabled = true;
  })
  ;

var postgres = builder.AddPostgres("postgres")
  .WithDataVolume()
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

var job_api = builder.AddGoApp(
    name: "job-api",
    appDirectory: "../services/job-api",
    packagePath: "./cmd/server"
  )
  // TODO: use make as resource dependency
  // .WithModTidy()
  .WithHttpEndpoint(targetPort: 9090, name: "grpc", env: "GRPC_PORT")
  .WithReference(db)
  .WithEnvironment(context => {
    context.EnvironmentVariables["POSTGRESDB_URI"] = db.Resource.UriExpression;
  })
  .WaitFor(db)
  .WaitForCompletion(migration)
  .PublishAsDockerComposeService((resource, service) => {
    service.Name = "job-api";
  });

// The aspireify skill will wire up your projects here.

builder.Build().Run();
