using System.Net;
using Microsoft.Extensions.Configuration;
using Aspire.Hosting;
using Aspire.Hosting.ApplicationModel;
using Aspire.Hosting.Go;
using Aspire.Hosting.Yarp;

namespace apphosts;

public sealed class JobWorkerBuilder(
    IDistributedApplicationBuilder builder
    , EndpointReference jobApiGrpcEndpoint
    , EndpointReference authGrpcEndpoint
    , EndpointReference demoContainerEndpoint
    , string workerClientId
    , string pubsubVerificationToken
)
{
    public EndpointReference Build(int port, bool isDebug)
    {
        var forwardProxy = ForwardProxy(demoContainerEndpoint);
        return Backend(port, isDebug, forwardProxy);
    }

    private EndpointReference Backend(
        int port
        , bool isDebug
        , EndpointReference forwardProxyEndpoint
    )
    {
        var workerClientSecret = builder.AddParameter("job-worker-client-secret", secret: true);
        var backend = builder.AddGoApp(
            name: "job-worker",
            appDirectory: "../services/job-worker",
            packagePath: "./cmd/worker"
        )
        .WithAppArgs("-worker", "3")
        .WithHttpEndpoint(
            name: "push",
            port: port,
            env: "PORT"
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
            context.EnvironmentVariables["LOCAL_PUBSUB_VERIFICATION_TOKEN"] = pubsubVerificationToken;
        })
        ;

        if (isDebug)
        {
            backend.WithDelveServer(new DelveServerOptions
            {
                Port = 2346,
                AcceptMultiClient = true,
                ContinueOnStart = true,
                OnlySameUser = false
            });
        }

        return backend.GetEndpoint("push", KnownNetworkIdentifiers.DefaultAspireContainerNetwork);
    }

    private EndpointReference ForwardProxy(EndpointReference demoContainerEndpoint)
    {
        return builder.AddDockerfile(
                name: "forward-proxy",
                contextPath: "../containers/forward-proxy"
            )
            .WithHttpEndpoint(
                name: "proxy",
                port: 3128,
                targetPort: 3128
            )
            .WithReference(demoContainerEndpoint)
            .WithEnvironment(context =>
            {
                context.EnvironmentVariables["SQUID_ALLOWED_DEMO_HOST_AND_PORT"] = demoContainerEndpoint.Property(EndpointProperty.HostAndPort);
            })
            .GetEndpoint("proxy")
        ;
    }
}
