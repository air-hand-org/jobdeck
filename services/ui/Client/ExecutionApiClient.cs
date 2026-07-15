using Grpc.Core;
using Jobdeck.Job.V1;

namespace ui.Client;

public sealed record ListExecutionsResult(List<Execution> Executions, string NextPageToken);

public sealed class ExecutionApiClient(ExecutionService.ExecutionServiceClient execs, ILogger<ExecutionApiClient> logger)
{
  private readonly ExecutionService.ExecutionServiceClient execs = execs;
  private readonly ILogger<ExecutionApiClient> logger = logger;

  public async Task<ApiResult> DoExecutionAsync(string jobId)
  {
    try
    {
      await execs.DoExecutionAsync(new DoExecutionRequest
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
      return ApiResult.Failure(ex.Status.Detail);
    }
    catch (RpcException ex)
    {
      logger.LogWarning(ex, "Execution API request failed");
      return ApiResult.Failure("Failed to call Execution API.");
    }
  }

  public async Task<ApiResult<ListExecutionsResult>> ListExecutionsAsync(string jobId, int pageSize, string pageToken = "")
  {
    try
    {
      var response = await execs.ListExecutionsAsync(new ListExecutionsRequest
      {
        JobId = jobId,
        PageSize = pageSize,
        PageToken = pageToken,
      });
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
      return ApiResult<ListExecutionsResult>.Failure(ex.Status.Detail);
    }
    catch (RpcException ex)
    {
      logger.LogWarning(ex, "Execution API request failed");
      return ApiResult<ListExecutionsResult>.Failure("Failed to call Execution API.");
    }
  }
}
