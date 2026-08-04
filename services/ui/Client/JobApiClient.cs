using System.Diagnostics;
using Grpc.Core;
using Jobdeck.Job.V1;
using ui.Auth;
using ui.Models;
using ui.Telemetry;

namespace ui.Client;

public sealed record ListJobsResult(IReadOnlyList<Job> Jobs, string NextPageToken);

public sealed class JobApiClient(JobService.JobServiceClient jobs, AccessTokenAccessor tokens, ILogger<JobApiClient> logger)
{
    public async Task<ApiResult<Job>> CreateJobAsync(HttpJobFormModel model, CancellationToken cancellationToken = default)
    {
        using var activity = UiTelemetry.Source.StartActivity("job.create", ActivityKind.Internal);
        activity?.SetTag("jobdeck.job.name", model.Name);

        try
        {
            var token = await tokens.GetAsync(cancellationToken)
                ?? throw new RpcException(new Status(StatusCode.Unauthenticated, "Authentication has expired."));

            var headers = new Metadata
            {
                {
                    "authorization", $"Bearer {token}"
                }
            };
            var response = await jobs.CreateJobAsync(
                new CreateJobRequest
                {
                    Name = model.Name,
                    Spec = model.ToJobSpec(),
                },
                headers,
                cancellationToken: cancellationToken
            );

            return ApiResult<Job>.Success(response.Job);
        }
        catch (RpcException ex)
        {
            logger.LogWarning(ex, "Job API request failed");
            activity?.SetStatus(ActivityStatusCode.Error, ex.Status.Detail);
            activity?.AddException(ex);
            return ApiResult<Job>.Failure(ApiError.FromRPCException(ex));
        }
    }

    public async Task<ApiResult<Job>> UpdateJobAsync(string jobId, HttpJobFormModel model, CancellationToken cancellationToken = default)
    {
        using var activity = UiTelemetry.Source.StartActivity("job.update", ActivityKind.Internal);
        activity?.SetTag("jobdeck.job.id", jobId);
        activity?.SetTag("jobdeck.job.name", model.Name);

        try
        {
            var token = await tokens.GetAsync(cancellationToken)
                ?? throw new RpcException(new Status(StatusCode.Unauthenticated, "Authentication has expired."));

            var headers = new Metadata
            {
                {
                    "authorization", $"Bearer {token}"
                }
            };
            var response = await jobs.UpdateJobAsync(
                new UpdateJobRequest
                {
                    JobId = jobId,
                    Name = model.Name,
                    Spec = model.ToJobSpec(),
                },
                headers,
                cancellationToken: cancellationToken
            );
            return ApiResult<Job>.Success(response.Job);
        }
        catch (RpcException ex)
        {
            logger.LogWarning(ex, "Job API request failed");
            activity?.SetStatus(ActivityStatusCode.Error, ex.Status.Detail);
            activity?.AddException(ex);
            return ApiResult<Job>.Failure(ApiError.FromRPCException(ex));
        }
    }

    public async Task<ApiResult> DeleteJobAsync(string jobId, CancellationToken cancellationToken = default)
    {
        using var activity = UiTelemetry.Source.StartActivity("job.delete", ActivityKind.Internal);
        activity?.SetTag("jobdeck.job.id", jobId);

        try
        {
            var token = await tokens.GetAsync(cancellationToken)
                ?? throw new RpcException(new Status(StatusCode.Unauthenticated, "Authentication has expired."));

            var headers = new Metadata
            {
                {
                    "authorization", $"Bearer {token}"
                }
            };
            var response = await jobs.DeleteJobAsync(
                new DeleteJobRequest
                {
                    JobId = jobId,
                },
                headers,
                cancellationToken: cancellationToken
            );
            return ApiResult.Success();
        }
        catch (RpcException ex)
        {
            logger.LogWarning(ex, "Job API request failed");
            activity?.SetStatus(ActivityStatusCode.Error, ex.Status.Detail);
            activity?.AddException(ex);
            return ApiResult.Failure(ApiError.FromRPCException(ex));
        }
    }

    public async Task<ApiResult<Job>> GetJobAsync(string jobId, CancellationToken cancellationToken = default)
    {
        using var activity = UiTelemetry.Source.StartActivity("job.get", ActivityKind.Internal);
        activity?.SetTag("jobdeck.job.id", jobId);

        try
        {
            var token = await tokens.GetAsync(cancellationToken)
                ?? throw new RpcException(new Status(StatusCode.Unauthenticated, "Authentication has expired."));

            var headers = new Metadata
            {
                {
                    "authorization", $"Bearer {token}"
                }
            };
            var response = await jobs.GetJobAsync(
                new GetJobRequest
                {
                    JobId = jobId,
                },
                headers,
                cancellationToken: cancellationToken
            );
            return ApiResult<Job>.Success(response.Job);
        }
        catch (RpcException ex)
        {
            logger.LogWarning(ex, "Job API request failed");
            activity?.SetStatus(ActivityStatusCode.Error, ex.Status.Detail);
            activity?.AddException(ex);
            return ApiResult<Job>.Failure(ApiError.FromRPCException(ex));
        }
    }

    public async Task<ApiResult<ListJobsResult>> ListJobsAsync(int pageSize, string pageToken = "", CancellationToken cancellationToken = default)
    {
        using var activity = UiTelemetry.Source.StartActivity("job.list", ActivityKind.Internal);
        activity?.SetTag("jobdeck.job.list_page_size", pageSize);

        try
        {
            var token = await tokens.GetAsync(cancellationToken)
                ?? throw new RpcException(new Status(StatusCode.Unauthenticated, "Authentication has expired."));

            var headers = new Metadata
            {
                {
                    "authorization", $"Bearer {token}"
                }
            };
            var response = await jobs.ListJobsAsync(
                new ListJobsRequest
                {
                    PageSize = pageSize,
                    PageToken = pageToken,
                },
                headers,
                cancellationToken: cancellationToken
            );
            return ApiResult<ListJobsResult>.Success(new ListJobsResult(
              [.. response.Jobs],
              response.NextPageToken
            ));
        }
        catch (RpcException ex)
        {
            logger.LogWarning(ex, "Job API request failed");
            activity?.SetStatus(ActivityStatusCode.Error, ex.Status.Detail);
            activity?.AddException(ex);
            return ApiResult<ListJobsResult>.Failure(ApiError.FromRPCException(ex));
        }
    }
}
