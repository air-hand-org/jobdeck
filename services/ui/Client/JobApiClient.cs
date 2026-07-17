using System.Diagnostics;
using Grpc.Core;
using Jobdeck.Job.V1;
using ui.Models;
using ui.Telemetry;

namespace ui.Client;

public sealed record ListJobsResult(IReadOnlyList<Job> Jobs, string NextPageToken);

public sealed class JobApiClient(JobService.JobServiceClient jobs, ILogger<JobApiClient> logger)
{
  private readonly JobService.JobServiceClient jobs = jobs;
  private readonly ILogger<JobApiClient> logger = logger;

  public async Task<ApiResult<Job>> CreateJobAsync(HttpJobFormModel model)
  {
    using var activity = UiTelemetry.Source.StartActivity("job.create", ActivityKind.Internal);
    activity?.SetTag("jobdeck.job.name", model.Name);

    try
    {
      var response = await jobs.CreateJobAsync(new CreateJobRequest
      {
        Name = model.Name,
        Spec = model.ToJobSpec(),
      });

      return ApiResult<Job>.Success(response.Job);
    }
    catch (RpcException ex) when (ex.StatusCode == StatusCode.InvalidArgument)
    {
      activity?.SetStatus(ActivityStatusCode.Error, ex.Status.Detail);
      activity?.AddException(ex);
      return ApiResult<Job>.Failure(ex.Status.Detail);
    }
    catch (RpcException ex)
    {
      logger.LogWarning(ex, "Job API request failed");
      activity?.SetStatus(ActivityStatusCode.Error, ex.Message);
      activity?.AddException(ex);
      return ApiResult<Job>.Failure("Failed to call Job API.");
    }
  }

  public async Task<ApiResult<Job>> UpdateJobAsync(string jobId, HttpJobFormModel model)
  {
    using var activity = UiTelemetry.Source.StartActivity("job.update", ActivityKind.Internal);
    activity?.SetTag("jobdeck.job.id", jobId);
    activity?.SetTag("jobdeck.job.name", model.Name);

    try
    {
      var response = await jobs.UpdateJobAsync(new UpdateJobRequest
      {
        JobId = jobId,
        Name = model.Name,
        Spec = model.ToJobSpec(),
      });
      return ApiResult<Job>.Success(response.Job);
    }
    catch (RpcException ex) when (
        ex.StatusCode == StatusCode.InvalidArgument
        ||
        ex.StatusCode == StatusCode.NotFound
    )
    {
      activity?.SetStatus(ActivityStatusCode.Error, ex.Status.Detail);
      activity?.AddException(ex);
      return ApiResult<Job>.Failure(ex.Status.Detail);
    }
    catch (RpcException ex)
    {
      logger.LogWarning(ex, "Job API request failed");
      activity?.SetStatus(ActivityStatusCode.Error, ex.Message);
      activity?.AddException(ex);
      return ApiResult<Job>.Failure("Failed to call Job API.");
    }
  }

  public async Task<ApiResult> DeleteJobAsync(string jobId)
  {
    using var activity = UiTelemetry.Source.StartActivity("job.delete", ActivityKind.Internal);
    activity?.SetTag("jobdeck.job.id", jobId);

    try
    {
      var response = await jobs.DeleteJobAsync(new DeleteJobRequest
      {
        JobId = jobId,
      });
      return ApiResult.Success();
    }
    catch (RpcException ex) when (
        ex.StatusCode == StatusCode.InvalidArgument
        ||
        ex.StatusCode == StatusCode.NotFound
    )
    {
      activity?.SetStatus(ActivityStatusCode.Error, ex.Status.Detail);
      activity?.AddException(ex);
      return ApiResult.Failure(ex.Status.Detail);
    }
    catch (RpcException ex)
    {
      logger.LogWarning(ex, "Job API request failed");
      activity?.SetStatus(ActivityStatusCode.Error, ex.Message);
      activity?.AddException(ex);
      return ApiResult.Failure("Failed to call Job API.");
    }
  }

  public async Task<ApiResult<Job>> GetJobAsync(string jobId)
  {
    using var activity = UiTelemetry.Source.StartActivity("job.get", ActivityKind.Internal);
    activity?.SetTag("jobdeck.job.id", jobId);

    try
    {
      var response = await jobs.GetJobAsync(new GetJobRequest
      {
        JobId = jobId,
      });
      return ApiResult<Job>.Success(response.Job);
    }
    catch (RpcException ex) when (
        ex.StatusCode == StatusCode.InvalidArgument
        ||
        ex.StatusCode == StatusCode.NotFound
    )
    {
      activity?.SetStatus(ActivityStatusCode.Error, ex.Status.Detail);
      activity?.AddException(ex);
      return ApiResult<Job>.Failure(ex.Status.Detail);
    }
    catch (RpcException ex)
    {
      logger.LogWarning(ex, "Job API request failed");
      activity?.SetStatus(ActivityStatusCode.Error, ex.Message);
      activity?.AddException(ex);
      return ApiResult<Job>.Failure("Failed to call Job API.");
    }
  }

  public async Task<ApiResult<ListJobsResult>> ListJobsAsync(int pageSize, string pageToken = "")
  {
    using var activity = UiTelemetry.Source.StartActivity("job.list", ActivityKind.Internal);
    activity?.SetTag("jobdeck.job.list_page_size", pageSize);

    try
    {
      var response = await jobs.ListJobsAsync(new ListJobsRequest
      {
        PageSize = pageSize,
        PageToken = pageToken,
      });
      return ApiResult<ListJobsResult>.Success(new ListJobsResult(
        [.. response.Jobs],
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
      return ApiResult<ListJobsResult>.Failure(ex.Status.Detail);
    }
    catch (RpcException ex)
    {
      logger.LogWarning(ex, "Job API request failed");
      activity?.SetStatus(ActivityStatusCode.Error, ex.Message);
      activity?.AddException(ex);
      return ApiResult<ListJobsResult>.Failure("Failed to call Job API.");
    }
  }

}
