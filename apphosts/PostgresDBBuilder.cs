using Aspire.Hosting;
using Aspire.Hosting.ApplicationModel;

namespace apphosts;

public sealed class PostgresDBBuilder(
    IDistributedApplicationBuilder builder
    , IResourceBuilder<PostgresServerResource> postgres
)
{
    private IResourceBuilder<ExecutableResource> initDB(string serviceName, IResourceBuilder<PostgresDatabaseResource> db)
    {
        var createFunctions = builder.AddGoApp(
            name: $"{serviceName}-create-db-functions",
            appDirectory: "../tools/postgres",
            packagePath: "./cmd/execsqlfile"
        )
        .WithArgs("-sqlfile", "../iac/app/sql/custom_role_functions.sql")
        .WithReference(db)
        .WithEnvironment("POSTGRESDB_URI", db.Resource.UriExpression)
        .WaitFor(db)
        ;

        builder.AddGoApp(
            name: $"{serviceName}-create-roles",
            appDirectory: "../tools/postgres",
            packagePath: "./cmd/createrole"
        )
        .WithArgs("-service", serviceName)
        .WithReference(db)
        .WithEnvironment("POSTGRESDB_URI", db.Resource.UriExpression)
        .WaitFor(db)
        .WaitForCompletion(createFunctions)
        ;
    }
}
