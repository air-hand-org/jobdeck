using System.Diagnostics;
using Grpc.Core;
using Jobdeck.Job.V1;
using ui.Auth;
using ui.Telemetry;

namespace ui.Client;

public sealed record ListExecutionsResult(List<Execution> Executions, string NextPageToken);

public sealed class ExecutionApiClient(ExecutionService.ExecutionServiceClient execs, AccessTokenAccessor tokens, ILogger<ExecutionApiClient> logger)
{
    public async Task<ApiResult> DoExecutionAsync(string jobId, CancellationToken cancellationToken = default)
    {
        using var activity = UiTelemetry.Source.StartActivity("execution.do", ActivityKind.Internal);
        activity?.SetTag("jobdeck.execution.job_id", jobId);

        try
        {
            var token = await tokens.GetAsync(cancellationToken);
            if (token is null)
            {
                throw new RpcException(new Status(StatusCode.Unauthenticated, "Authentication has expired."));
            }

            var headers = new Metadata
            {
                {
                    "authorization", $"Bearer {token}"
                }
            };
            await execs.DoExecutionAsync(
                new DoExecutionRequest
                {
                    JobId = jobId,
                },
                headers,
                cancellationToken: cancellationToken
            );

            return ApiResult.Success();
        }
        catch (RpcException ex) when (
            ex.StatusCode == StatusCode.InvalidArgument
            ||
            ex.StatusCode == StatusCode.NotFound
            ||
            ex.StatusCode == StatusCode.Unauthenticated
        )
        {
            activity?.SetStatus(ActivityStatusCode.Error, ex.Status.Detail);
            activity?.AddException(ex);
            return ApiResult.Failure(ex.Status.Detail);
        }
        catch (RpcException ex)
        {
            logger.LogWarning(ex, "Execution API request failed");
            activity?.SetStatus(ActivityStatusCode.Error, ex.Message);
            activity?.AddException(ex);
            return ApiResult.Failure("Failed to call Execution API.");
        }
    }

    public async Task<ApiResult<ListExecutionsResult>> ListExecutionsAsync(string jobId, int pageSize, string pageToken = "", CancellationToken cancellationToken = default)
    {
        using var activity = UiTelemetry.Source.StartActivity("execution.list", ActivityKind.Internal);
        activity?.SetTag("jobdeck.execution.job_id", jobId);

        try
        {
            var token = await tokens.GetAsync(cancellationToken);
            if (token is null)
            {
                throw new RpcException(new Status(StatusCode.Unauthenticated, "Authentication has expired."));
            }

            var headers = new Metadata
            {
                {
                    "authorization", $"Bearer {token}"
                }
            };
            var response = await execs.ListExecutionsAsync(
                new ListExecutionsRequest
                {
                    JobId = jobId,
                    PageSize = pageSize,
                    PageToken = pageToken,
                },
                headers,
                cancellationToken: cancellationToken
            );
            return ApiResult<ListExecutionsResult>.Success(new ListExecutionsResult(
              [.. response.Executions],
              response.NextPageToken
            ));
        }
        catch (RpcException ex) when (
            ex.StatusCode == StatusCode.InvalidArgument
            ||
            ex.StatusCode == StatusCode.NotFound
        )
        {
            activity?.SetStatus(ActivityStatusCode.Error, ex.Status.Detail);
            activity?.AddException(ex);
            return ApiResult<ListExecutionsResult>.Failure(ex.Status.Detail);
        }
        catch (RpcException ex)
        {
            logger.LogWarning(ex, "Execution API request failed");
            activity?.SetStatus(ActivityStatusCode.Error, ex.Message);
            activity?.AddException(ex);
            return ApiResult<ListExecutionsResult>.Failure("Failed to call Execution API.");
        }
    }
}
