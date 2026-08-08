using ui.Auth;
using ui.Components;
using ui.Client;
using ui.Session;
using ui.Telemetry;
using Microsoft.AspNetCore.Authentication.Cookies;
using Microsoft.AspNetCore.Diagnostics.HealthChecks;
using Microsoft.Extensions.Diagnostics.HealthChecks;
using Jobdeck.Auth.V1;
using Jobdeck.Job.V1;
using Grpc.Health.V1;
using OpenTelemetry.Resources;
using OpenTelemetry.Trace;

var builder = WebApplication.CreateBuilder(args);

// Add services to the container.
builder.Services.AddRazorComponents()
  .AddInteractiveServerComponents()
  ;
builder.Services.AddOpenTelemetry()
  .ConfigureResource(resource => resource.AddService("jobdeck-ui"))
  .WithTracing(tracing =>
  {
      tracing
        .AddAspNetCoreInstrumentation()
        .AddGrpcClientInstrumentation()
        .AddSource(UiTelemetry.ActivitySourceName);

      if (!string.IsNullOrWhiteSpace(builder.Configuration["OTEL_EXPORTER_OTLP_ENDPOINT"]))
      {
          tracing.AddOtlpExporter();
      }
  })
  ;
builder.Services.AddGrpcClient<JobService.JobServiceClient>(options =>
  {
      var address = builder.Configuration["JOB_API_GRPC_ENDPOINT"] ?? throw new InvalidOperationException("JOB_API_GRPC_ENDPOINT is required");
      options.Address = new Uri(address);
  }
);
builder.Services.AddGrpcClient<ExecutionService.ExecutionServiceClient>(options =>
  {
      var address = builder.Configuration["JOB_API_GRPC_ENDPOINT"] ?? throw new InvalidOperationException("JOB_API_GRPC_ENDPOINT is required");
      options.Address = new Uri(address);
  }
);
builder.Services.AddGrpcClient<AuthService.AuthServiceClient>(options =>
  {
      var address = builder.Configuration["AUTH_GRPC_ENDPOINT"] ?? throw new InvalidOperationException("AUTH_GRPC_ENDPOINT is required");
      options.Address = new Uri(address);
  }
);
builder.Services.AddScoped<JobApiClient>();
builder.Services.AddScoped<ExecutionApiClient>();
builder.Services.AddScoped<AuthApiClient>();
builder.Services.AddScoped<UICookieAuthenticationEvents>();

builder.Services.AddGrpcClient<Health.HealthClient>(
    "job-api-health",
    options =>
    {
        var address = builder.Configuration["JOB_API_GRPC_ENDPOINT"] ?? throw new InvalidOperationException("JOB_API_GRPC_ENDPOINT is required");
        options.Address = new Uri(address);
    }
);
builder.Services.AddGrpcClient<Health.HealthClient>(
    "auth-health",
    options =>
    {
        var address = builder.Configuration["AUTH_GRPC_ENDPOINT"] ?? throw new InvalidOperationException("AUTH_GRPC_ENDPOINT is required");
        options.Address = new Uri(address);
    }
);
builder.Services.AddHealthChecks()
    .AddTypeActivatedCheck<HealthCheckClient>(
        "job-api",
        failureStatus: HealthStatus.Unhealthy,
        tags: ["ready"],
        args: ["job-api-health"]
    )
    .AddTypeActivatedCheck<HealthCheckClient>(
        "auth",
        failureStatus: HealthStatus.Unhealthy,
        tags: ["ready"],
        args: ["auth-health"]
    )
    ;

builder.Services.AddAuthentication(CookieAuthenticationDefaults.AuthenticationScheme)
    .AddCookie(options =>
        {
            options.EventsType = typeof(UICookieAuthenticationEvents);
            options.LoginPath = "/account/login";
            options.LogoutPath = "/account/logout";
            options.Cookie.HttpOnly = true;
            options.Cookie.SecurePolicy = CookieSecurePolicy.Always;
            options.Cookie.SameSite = SameSiteMode.Lax;
            options.SlidingExpiration = false;
        }
    );
builder.Services.AddAuthorization();
builder.Services.AddCascadingAuthenticationState();

builder.Services.AddMemoryCache();
builder.Services.AddSingleton<IUISessionStore, MemoryUISessionStore>();
builder.Services.AddScoped<AccessTokenAccessor>();
builder.Services.AddScoped<UIAuthenticationService>();


var app = builder.Build();

// Configure the HTTP request pipeline.
if (!app.Environment.IsDevelopment())
{
    app.UseExceptionHandler("/Error", createScopeForErrors: true);
    // The default HSTS value is 30 days. You may want to change this for production scenarios, see https://aka.ms/aspnetcore-hsts.
    app.UseHsts();
}
app.UseStatusCodePagesWithReExecute("/not-found", createScopeForStatusCodePages: true);
app.UseHttpsRedirection();

app.UseAuthentication();
app.UseAuthorization();
app.UseAntiforgery();

app.MapHealthChecks(
    "/health/live",
    new HealthCheckOptions
    {
        Predicate = _ => false,
    }
);
app.MapHealthChecks(
    "/health/ready",
    new HealthCheckOptions
    {
        Predicate = check => check.Tags.Contains("ready"),
    }
);
app.MapStaticAssets();
app.MapRazorComponents<App>()
    .AddInteractiveServerRenderMode();

app.Run();
