using System.Net;
using Microsoft.Extensions.Configuration;
using Aspire.Hosting;
using Aspire.Hosting.ApplicationModel;

namespace apphosts;

public sealed class JobWorkerBuilder(
    IDistributedApplicationBuilder builder
    , EndpointReference jobApiGrpcEndpoint
    , EndpointReference authGrpcEndpoint
    , EndpointReference demoContainerEndpoint
    , string workerClientId
)
{
    public void Build(bool isDebug)
    {
        var forwardProxy = ForwardProxy(demoContainerEndpoint);
        Worker(isDebug, forwardProxy);
    }

    private void Worker(
        bool isDebug
        , IResourceBuilder<ContainerResource> forwardProxy
    )
    {
        var workerClientSecret = builder.AddParameter("job-worker-client-secret", secret: true);
        var forwardProxyEndpoint = forwardProxy.GetEndpoint("proxy");
        // TODO: use WithDelveServer after v13.5
        if (isDebug)
        {
            var backend = builder.AddExecutable(
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
              .WaitFor(forwardProxy)
              ;
        }
        else
        {
            var backend = builder.AddGoApp(
                name: "job-worker",
                appDirectory: "../services/job-worker",
                packagePath: "./cmd/worker"
              )
              .WithAppArgs("-worker", "3")
              //    .WithDelveServer(port: 2346) // --only-same-user=false needed
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
              .WaitFor(forwardProxy)
              ;
        }
    }

    private IResourceBuilder<ContainerResource> ForwardProxy(EndpointReference demoContainerEndpoint)
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
            .WithEnvironment(context =>
            {
                context.EnvironmentVariables["SQUID_ALLOWED_DEMO_HOST_AND_PORT"] = demoContainerEndpoint.Property(EndpointProperty.HostAndPort);
            })
        ;
    }
}
