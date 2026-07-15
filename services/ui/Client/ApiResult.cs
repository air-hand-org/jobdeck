namespace ui.Client;

public sealed record ApiResult<T>(T? Value, string? Error)
{
  public bool IsSuccess => Error is null;

  public static ApiResult<T> Success(T value) => new(Value: value, Error: null);
  public static ApiResult<T> Failure(string error) => new(Value: default, Error: error);
}

public sealed record ApiResult(string? Error)
{
  public bool IsSuccess => Error is null;
  public static ApiResult Success() => new(Error: null);
  public static ApiResult Failure(string error) => new(error);
}
