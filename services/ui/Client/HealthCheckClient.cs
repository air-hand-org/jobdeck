using Microsoft.Extensions.Diagnostics.HealthChecks;
using Grpc.Core;
using Grpc.Net.ClientFactory;
using Grpc.Health.V1;

namespace ui.Client;

public sealed class HealthCheckClient(GrpcClientFactory clientFactory, string serviceName) : IHealthCheck
{
    public async Task<HealthCheckResult> CheckHealthAsync(
        HealthCheckContext context,
        CancellationToken cancellationToken = default
    )
    {
        try
        {
            var healthClient = clientFactory.CreateClient<Health.HealthClient>(serviceName);
            var response = await healthClient.CheckAsync(
                new HealthCheckRequest
                {
                    Service = "",
                },
                cancellationToken: cancellationToken
            );

            if (response.Status == HealthCheckResponse.Types.ServingStatus.Serving)
            {
                return HealthCheckResult.Healthy();
            }
            else
            {
                return HealthCheckResult.Unhealthy($"{serviceName}: {response.Status}");
            }
        }
        catch (Exception ex)
        {
            return HealthCheckResult.Unhealthy($"{serviceName} is unavailable", ex);
        }
    }
}
