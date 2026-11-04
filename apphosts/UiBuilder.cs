using Aspire.Hosting;
using Aspire.Hosting.ApplicationModel;
using Aspire.Hosting.Yarp;
using Aspire.Hosting.Yarp.Transforms;
using Aspire.Hosting.MongoDB;

namespace apphosts;

public sealed class UiBuilder(
    IDistributedApplicationBuilder builder,
    EndpointReference jobApiGrpcEndpoint,
    EndpointReference authGrpcEndpoint,
    IResourceBuilder<MongoDBDatabaseResource> mongodb,
    string imageTag,
    string repoRoot,
    bool useLocal
)
{
    private const string Image = "ghcr.io/air-hand-org/jobdeck-ui";

    private readonly string? localProjectPath = GetLocalProjectPath(repoRoot, useLocal);

    public IResourceBuilder<YarpResource> Build()
    {
        var backend = localProjectPath is null
            ? ContainerBackend()
            : LocalBackend();

        return builder.AddYarp("ui")
            .WithExternalHttpEndpoints()
            .WithHostHttpsPort(port: 7055)
            .WithConfiguration(yarp => yarp.AddRoute(backend)
                .WithTransformUseOriginalHostHeader(useOriginal: true)
                .WithTransformRequestHeader("X-Forwarded-Proto", "https", append: false))
            .WithHttpHealthCheck("/health/ready");
    }

    private EndpointReference ContainerBackend()
    {
        return ConfigureBackend(builder.AddContainer(
                name: "ui-backend",
                image: Image,
                tag: imageTag
            )
            .WithHttpEndpoint(
                name: "http",
                targetPort: 8080
            ))
            .GetEndpoint("http");
    }

    private EndpointReference LocalBackend()
    {
        return ConfigureBackend(builder.AddProject(
                name: "ui-backend",
                projectPath: localProjectPath!,
                configure: options => options.ExcludeLaunchProfile = true
            )
            .WithHttpEndpoint(
                name: "http",
                env: "ASPNETCORE_HTTP_PORTS"
            )
            .WithEnvironment("ASPNETCORE_ENVIRONMENT", "Development"))
            .GetEndpoint("http");
    }

    private IResourceBuilder<T> ConfigureBackend<T>(IResourceBuilder<T> backend)
        where T : IResource, IResourceWithEnvironment, IResourceWithEndpoints
    {
        var githubAppOAuthClientID = builder.AddParameter("github-oauth-client-id", secret: false);
        var githubAppOAuthClientSecret = builder.AddParameter("github-oauth-client-secret", secret: true);
        return backend
            .WithReference(jobApiGrpcEndpoint)
            .WithReference(authGrpcEndpoint)
            .WithReference(mongodb)
            .WithEnvironment("ASPNETCORE_FORWARDEDHEADERS_ENABLED", "true")
            .WithEnvironment(context =>
            {
                context.EnvironmentVariables["JOB_API_GRPC_ENDPOINT"] = jobApiGrpcEndpoint.Property(EndpointProperty.Url);
                context.EnvironmentVariables["AUTH_GRPC_ENDPOINT"] = authGrpcEndpoint.Property(EndpointProperty.Url);
                context.EnvironmentVariables["MONGODB_URI"] = mongodb.Resource.ConnectionStringExpression;
                context.EnvironmentVariables["MONGODB_DATABASE"] = mongodb.Resource.DatabaseName;

                context.EnvironmentVariables["GITHUB_OAUTH_CLIENT_ID"] = githubAppOAuthClientID;
                context.EnvironmentVariables["GITHUB_OAUTH_CLIENT_SECRET"] = githubAppOAuthClientSecret;
            })
            .WithHttpHealthCheck("/health/ready");
    }

    private static string? GetLocalProjectPath(string repoRoot, bool useLocal)
    {
        if (!useLocal)
        {
            return null;
        }

        var projectPath = Path.Combine(repoRoot, "services", "ui", "src", "ui.csproj");
        if (!File.Exists(projectPath))
        {
            return null;
        }

        return projectPath;
    }
}
