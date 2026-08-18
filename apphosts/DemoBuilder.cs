using System.Net;
using Microsoft.Extensions.Configuration;
using Aspire.Hosting;
using Aspire.Hosting.ApplicationModel;
using Aspire.Hosting.Yarp;

namespace apphosts;

public sealed class DemoBuilder(
    IDistributedApplicationBuilder builder
)
{
    public EndpointReference Build(int port, NetworkIdentifier networkId)
    {
        var backend = Backend();
        return ReverseProxy(port, backend, networkId);
    }

    private EndpointReference ReverseProxy(int port, EndpointReference backend, NetworkIdentifier networkId)
    {
        var yarp = builder.AddYarp("demo")
            .WithExternalHttpEndpoints()
            .WithHostHttpsPort(port)
            .WithConfiguration(yarp =>
            {
                yarp.AddRoute(backend);
            })
            ;
        return yarp.GetEndpoint("https", networkId);
    }

    private EndpointReference Backend()
    {
        return builder.AddGoApp(
            name: "demo-backend",
            appDirectory: "../services/demo",
            packagePath: "./cmd/server"
        )
        .WithHttpEndpoint(
            name: "http",
            env: "PORT"
        )
        .GetEndpoint("http")
        ;
    }
}
