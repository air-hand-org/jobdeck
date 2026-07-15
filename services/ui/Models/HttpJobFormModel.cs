using System.ComponentModel.DataAnnotations;
using Google.Protobuf.WellKnownTypes;
using Jobdeck.Job.V1;
using ui.Validation;

using JobdeckHttpMethod = Jobdeck.Job.V1.HttpMethod;
using JStatusCodes = Jobdeck.Job.V1.StatusCodes;

namespace ui.Models;

public sealed class HttpJobFormModel : IValidatableObject
{
  [Required]
  public string? Name { get; set; }

  [Required]
  [Url]
  public string? Url {get; set;}

  [NotUnspecifiedEnum(ErrorMessage = "HTTP method is required.")]
  public JobdeckHttpMethod Method {get; set;} = JobdeckHttpMethod.Unspecified;

  public List<int> ExpectedStatusCodes { get; set; } = [200];

  [Required]
  public long TimeoutSeconds { get; set; }

  [Required]
  public bool FollowRedirects { get; set; }

  [Required]
  public bool RetryOnTimeout {get; set;}

  [Required]
  public bool RetryOnNetworkError { get; set; }

  public List<int> RetryOnStatusCodes { get; set; } = [];

  [Required]
  public int MaxAttempts { get; set; } = 1;

  [Required]
  public long InitialDelaySeconds { get; set; } = 0;

  [Required]
  public long MaxDelaySeconds { get; set; } = 0;

  public IEnumerable<ValidationResult> Validate(ValidationContext context)
  {
    if (ExpectedStatusCodes.Count == 0)
    {
      yield return new ValidationResult("Expected status codes is required.", [nameof(ExpectedStatusCodes)]);
    }

    if (ExpectedStatusCodes.Any(code => code is < 100 or > 599))
    {
      yield return new ValidationResult("Expected status codes must be between 100 and 599.", [nameof(ExpectedStatusCodes)]);
    }

    if (RetryOnStatusCodes.Any(code => code is < 100 or > 599))
    {
      yield return new ValidationResult("Retry status codes must be between 100 and 599.", [nameof(RetryOnStatusCodes)]);
    }
  }

  public JobSpec ToJobSpec()
  {
    return new JobSpec
    {
      Type = JobType.HttpRequest,
      Http = new HttpJobConfig
      {
        Url = Url!,
        Method = Method,
        ExpectedStatusCodes = new JStatusCodes
        {
          Codes = { ExpectedStatusCodes }
        },
        Timeout = Duration.FromTimeSpan(TimeSpan.FromSeconds(TimeoutSeconds)),
        FollowRedirects = FollowRedirects,
        RetryOnTimeout = RetryOnTimeout,
        RetryOnNetworkError = RetryOnNetworkError,
        RetryOnStatusCodes = new JStatusCodes
        {
          Codes = { RetryOnStatusCodes },
        },
      },
      RetryPolicy = new RetryPolicy
      {
        MaxAttempts = MaxAttempts,
        BackoffType = BackoffType.Fixed,
        InitialDelay = Duration.FromTimeSpan(TimeSpan.FromSeconds(InitialDelaySeconds)),
        MaxDelay = Duration.FromTimeSpan(TimeSpan.FromSeconds(MaxDelaySeconds)),
      }
    };
  }

  public static HttpJobFormModel FromProto(Job job)
  {
    if (job.Spec?.Http is null)
    {
      throw new ArgumentException("Job must be an HTTP job.", nameof(job));
    }

    var http = job.Spec.Http;
    var retryPolicy = job.Spec.RetryPolicy ?? new RetryPolicy();

    return new HttpJobFormModel
    {
      Name = job.Name,
      Url = http.Url,
      Method = http.Method,
      ExpectedStatusCodes = http.ExpectedStatusCodes?.Codes.ToList() ?? [],
      TimeoutSeconds = (long)(http.Timeout?.ToTimeSpan().TotalSeconds ?? 0),
      FollowRedirects = http.FollowRedirects,
      RetryOnTimeout = http.RetryOnTimeout,
      RetryOnNetworkError = http.RetryOnNetworkError,
      RetryOnStatusCodes = http.RetryOnStatusCodes?.Codes.ToList() ?? [],
      MaxAttempts = retryPolicy.MaxAttempts,
      InitialDelaySeconds = (long)(retryPolicy.InitialDelay?.ToTimeSpan().TotalSeconds ?? 0),
      MaxDelaySeconds = (long)(retryPolicy.MaxDelay?.ToTimeSpan().TotalSeconds ?? 0),
    };
  }
}
