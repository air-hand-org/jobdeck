using ui.Components;
using ui.Client;
using Grpc.Net.Client;
using Jobdeck.Job.V1;

var builder = WebApplication.CreateBuilder(args);

// Add services to the container.
builder.Services.AddRazorComponents()
  .AddInteractiveServerComponents()
  ;
builder.Services.AddSingleton(sp => {
  var address = builder.Configuration["JOB_API_GRPC_ENDPOINT"] ?? throw new InvalidOperationException("JOB_API_GRPC_ENDPOINT is required");
  return GrpcChannel.ForAddress(address);
});
builder.Services.AddSingleton(sp => {
  var channel = sp.GetRequiredService<GrpcChannel>();
  return new JobService.JobServiceClient(channel);
});
builder.Services.AddSingleton(sp => {
  var channel = sp.GetRequiredService<GrpcChannel>();
  return new ExecutionService.ExecutionServiceClient(channel);
});
builder.Services.AddScoped<JobApiClient>();
builder.Services.AddScoped<ExecutionApiClient>();

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

app.UseAntiforgery();

app.MapStaticAssets();
app.MapRazorComponents<App>()
    .AddInteractiveServerRenderMode();

app.Run();
