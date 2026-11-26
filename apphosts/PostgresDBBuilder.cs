using Aspire.Hosting;
using Aspire.Hosting.ApplicationModel;

namespace apphosts;

public sealed class TempFile : IDisposable
{
    private readonly string Path { get; init; }

    public TempFile()
    {
        Path = Path.GetTempFileName();
    }

    public void Dispose()
    {
        File.Delete(Path);
    }
}

public sealed record PostgresUser(string User, string Password);

public sealed class PostgresDBResource(
    IResourceBuilder<PostgresDatabaseResource> db
    , IResourceBuilder<ExecutableResource> provisioner
    , PostgresUser runtimeUser
    , PostgresUser migrateUser
)
{
    public readonly IResourceBuilder<PostgresDataabaseResource> DB { get; init; } = db;
    public readonly IResourceBuilder<ExecutableResource> Provisioner { get; init; } = provisioner;

    public ReferenceExpression UriForRuntime() => UriForLogin(runtimeUser);
    public ReferenceExpression UriForMigrate() => UriForLogin(migrateUser);

    private ReferenceExpression UriForLogin(PostgresUser user)
    {
        var server = db.Resource.Parent;
        return ReferenceExpression.Create(
            $"postgresql://{user.User:uri}:{user.Password:uri}@{server.Host}:{server.Port}/{db.Resource.DatabaseName:uri}"
        );
    }
}

public sealed class PostgresDBBuilder(
    IDistributedApplicationBuilder builder
    , IResourceBuilder<PostgresServerResource> postgres
)
{
    public PostgresDBResource ProvisionDB(string serviceName)
    {
        var db = postgres.AddDatabase(serviceName);
        return InitDB(serviceName, db);
    }

    private PostgresDBResource InitDB(string serviceName, IResourceBuilder<PostgresDatabaseResource> db)
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

        var createRoles = builder.AddGoApp(
            name: $"{serviceName}-create-db-roles",
            appDirectory: "../tools/postgres",
            packagePath: "./cmd/createrole"
        )
        .WithArgs("-service", serviceName)
        .WithReference(db)
        .WithEnvironment("POSTGRESDB_URI", db.Resource.UriExpression)
        .WaitFor(db)
        .WaitForCompletion(createFunctions)
        ;

        var runtimeUser = new PostgresUser($"{serviceName}-runtime-user", string.Empty);
        var migrateUser = new PostgresUser($"{serviceName}-migrate-user", string.Empty);

        using var temp = new TempFile();
        using var file = File.CreateText(temp.Path);
        file.WriteLine($"CREATE USER {runtimeUser.User} WITH ROLE {serviceName}-runtime;");
        file.WriteLine($"CREATE USER {migrateUser.User} WITH ROLE {serviceName}-migrator;");

        var createUsers = builder.AddGoApp(
            name: $"{serviceName}-create-db-users",
            appDirectory: "../tools/postgres",
            packagePath: "./cmd/execsqlfile"
        )
        .WithArgs("-sqlfile", temp.Path)
        .WithReference(db)
        .WithEnvironment("POSTGRESDB_URI", db.Resource.UriExpression)
        .WaitFor(db)
        .WaitForCompletion(createRoles)
        ;

        return new PostgresDBResource(
            db,
            createUsers,
            runtimeUser,
            migrateUser
        );
    }
}
