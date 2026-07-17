using System.Diagnostics;
using Grpc.Core;
using Jobdeck.Auth.V1;
using ui.Models;
using ui.Telemetry;

namespace ui.Client;

public sealed class AuthApiClient(AuthService.AuthServiceClient auth, ILogger<AuthApiClient> logger)
{
    public async Task<ApiResult<Token>> AuthenticateLocalClientAsync(LoginFormModel model, CancellationToken cancellationToken = default)
    {
        using var activity = UiTelemetry.Source.StartActivity("auth.login", ActivityKind.Internal);

        try
        {
            var response = await auth.AuthenticateLocalClientAsync(
                new AuthenticateLocalClientRequest
                {
                    ClientId = model.Id,
                    ClientSecret = model.Password,
                    Audience = "jobdeck-job-api",
                },
                cancellationToken: cancellationToken
            );

            return ApiResult<Token>.Success(response.Token);
        }
        catch (RpcException ex) when (ex.StatusCode == StatusCode.InvalidArgument)
        {
            activity?.SetStatus(ActivityStatusCode.Error, ex.Status.Detail);
            activity?.AddException(ex);
            return ApiResult<Token>.Failure(ex.Status.Detail);
        }
        catch (RpcException ex)
        {
            logger.LogWarning(ex, "Auth API request failed");
            activity?.SetStatus(ActivityStatusCode.Error, ex.Message);
            activity?.AddException(ex);
            return ApiResult<Token>.Failure("Failed to call Auth API.");
        }
    }
}
