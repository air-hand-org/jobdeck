using System.Security.Cryptography.X509Certificates;
using Aspire.Hosting;
using Aspire.Hosting.ApplicationModel;
using Aspire.Hosting.Yarp;
using Aspire.Hosting.Yarp.Transforms;

namespace apphosts;

public sealed class UiBuilder(
    IDistributedApplicationBuilder builder,
    EndpointReference jobApiGrpcEndpoint,
    EndpointReference authGrpcEndpoint,
    IResourceBuilder<PostgresServerResource> postgres,
    string imageTag,
    string repoRoot,
    bool useLocal
)
{
    private const string Image = "ghcr.io/air-hand-org/jobdeck-ui";

    private readonly string? localProjectPath = GetLocalProjectPath(repoRoot, useLocal);

    public IResourceBuilder<YarpResource> Build()
    {
        var db = postgres.AddDatabase("jobdeck-ui");
        var migrator = Migrator(db);

        var backend = localProjectPath is null
            ? ContainerBackend(db, migrator)
            : LocalBackend(db, migrator);

        return builder.AddYarp("ui")
            .WithExternalHttpEndpoints()
            .WithHostHttpsPort(port: 7055)
            .WithConfiguration(yarp => yarp.AddRoute(backend)
                .WithTransformUseOriginalHostHeader(useOriginal: true)
                .WithTransformRequestHeader("X-Forwarded-Proto", "https", append: false))
            .WithHttpHealthCheck("/health/ready");
    }

    private EndpointReference ContainerBackend<T, U>(
        IResourceBuilder<T> db
        , IResourceBuilder<U> migrator
    )
        where T : class, IResource, IResourceWithConnectionString
        where U : class, IResource
    {
        return ConfigureBackend(
                builder.AddContainer(
                    name: "ui-backend",
                    image: Image,
                    tag: imageTag
                )
                .WithHttpEndpoint(
                    name: "http",
                    targetPort: 8080
                )
                , db
                , migrator
            )
            .GetEndpoint("http");
    }

    private EndpointReference LocalBackend<T, U>(
        IResourceBuilder<T> db
        , IResourceBuilder<U> migrator
    )
        where T : class, IResource, IResourceWithConnectionString
        where U : class, IResource
    {
        return ConfigureBackend(
                builder.AddProject(
                    name: "ui-backend",
                    projectPath: localProjectPath!,
                    configure: options => options.ExcludeLaunchProfile = true
                )
                .WithHttpEndpoint(
                    name: "http",
                    env: "ASPNETCORE_HTTP_PORTS"
                )
                .WithEnvironment("ASPNETCORE_ENVIRONMENT", "Development")
                , db
                , migrator
            )
            .GetEndpoint("http");
    }

    private IResourceBuilder<T> ConfigureBackend<T, U, V>(
        IResourceBuilder<T> backend
        , IResourceBuilder<U> db
        , IResourceBuilder<V> migrator
    )
        where T : IResource, IResourceWithEnvironment, IResourceWithEndpoints, IResourceWithWaitSupport
        where U : class, IResource, IResourceWithConnectionString
        where V : class, IResource
    {
        var githubAppOAuthClientID = builder.AddParameter("github-oauth-client-id", secret: false);
        var githubAppOAuthClientSecret = builder.AddParameter("github-oauth-client-secret", secret: true);
        var certFactory = new LocalCertificateParameterFactory(builder, string.Empty, builder.Configuration["UI_CERT_DIR"]);
        var dataProtectionCert = certFactory.AddParameterAsBase64("ui-dataprotection", X509KeyUsageFlags.KeyEncipherment);
        var openidEncryptionCert = certFactory.AddParameterAsBase64("ui-openid-encryption", X509KeyUsageFlags.KeyEncipherment);
        var openidSigningCert = certFactory.AddParameterAsBase64("ui-openid-signing", X509KeyUsageFlags.DigitalSignature);
        return backend
            .WithReference(jobApiGrpcEndpoint)
            .WithReference(authGrpcEndpoint)
            .WithReference(db)
            .WithEnvironment("ASPNETCORE_FORWARDEDHEADERS_ENABLED", "true")
            .WithEnvironment(context =>
            {
                context.EnvironmentVariables["JOB_API_GRPC_ENDPOINT"] = jobApiGrpcEndpoint.Property(EndpointProperty.Url);
                context.EnvironmentVariables["AUTH_GRPC_ENDPOINT"] = authGrpcEndpoint.Property(EndpointProperty.Url);
                context.EnvironmentVariables["ConnectionStrings__Postgres"] = db.Resource.ConnectionStringExpression;

                context.EnvironmentVariables["GITHUB_OAUTH_CLIENT_ID"] = githubAppOAuthClientID;
                context.EnvironmentVariables["GITHUB_OAUTH_CLIENT_SECRET"] = githubAppOAuthClientSecret;

                context.EnvironmentVariables["CERT_DATAPROTECTION_BASE64"] = dataProtectionCert;
                context.EnvironmentVariables["CERT_OPENID_ENCRYPTION_BASE64"] = openidEncryptionCert;
                context.EnvironmentVariables["CERT_OPENID_SIGNING_BASE64"] = openidSigningCert;
            })
            .WaitForCompletion(migrator)
            .WithHttpHealthCheck("/health/ready");
    }

    private IResourceBuilder<ContainerResource> Migrator<T>(
        IResourceBuilder<T> db
    )
        where T : class, IResource, IResourceWithConnectionString
    {
        var migrator = localProjectPath is null ?
            builder.AddContainer(
                name: "ui-db-migrator",
                image: "ghcr.io/air-hand-org/jobdeck-ui-migrator",
                tag: imageTag
            )
            :
            builder.AddDockerfile(
                name: "ui-db-migrator",
                contextPath: Path.GetFullPath(Path.Combine(repoRoot, "services", "ui")),
                stage: "migrator"
            )
            .WithBuildSecret("github_packages_token", builder.AddParameterFromConfiguration("github-packages-token", "GITHUB_PACKAGES_TOKEN", secret: true))
            .WithBuildArg("GITHUB_ACTOR", builder.AddParameterFromConfiguration("github-actor", "GITHUB_ACTOR"))
            ;

        return migrator
            .WithReference(db)
            .WithEnvironment("ConnectionStrings__Postgres", db.Resource.ConnectionStringExpression)
            .WithEnvironment("MIGRATOR_EXECUTE", "true")
            .WaitFor(db)
            ;
    }

    private static string? GetLocalProjectPath(string repoRoot, bool useLocal)
    {
        if (!useLocal)
        {
            return null;
        }

        var projectPath = Path.Combine(repoRoot, "services", "ui", "src", "bff", "ui.csproj");
        if (!File.Exists(projectPath))
        {
            // Keep released checkouts usable while the BFF layout is adopted.
            projectPath = Path.Combine(repoRoot, "services", "ui", "src", "ui.csproj");
        }
        if (!File.Exists(projectPath))
        {
            return null;
        }

        return projectPath;
    }
}
