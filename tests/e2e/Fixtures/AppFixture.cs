using Aspire.Hosting;
using Aspire.Hosting.Testing;
using Microsoft.Extensions.Logging;
using Xunit;

namespace e2e.Fixtures;

[CollectionDefinition(nameof(AppFixture))]
public sealed class AppCollection : ICollectionFixture<AppFixture>;

public sealed class AppFixture : IAsyncLifetime
{
    public UITestTarget Target { get; private set; } = null!;

    private DistributedApplication? app;

    public async Task InitializeAsync()
    {
        var baseUrl = Environment.GetEnvironmentVariable("E2E_BASE_URL");
        if (!string.IsNullOrWhiteSpace(baseUrl))
        {
            if (!Uri.TryCreate(baseUrl, UriKind.Absolute, out var uri)
                || (uri.Scheme is not "http" and not "https")
            )
            {
                throw new InvalidOperationException("E2E_BASE_URL must be an absolute HTTP(S) URL.");
            }
            Target = new(uri);
            return;
        }

        using var cts = new CancellationTokenSource(TimeSpan.FromMinutes(5));

        var appHost = await DistributedApplicationTestingBuilder.CreateAsync<Projects.Apphosts>(cts.Token);
        appHost.Services.AddLogging(logging =>
        {
            logging.SetMinimumLevel(LogLevel.Debug);
            // Override the logging filters from the app's configuration
            logging.AddFilter(appHost.Environment.ApplicationName, LogLevel.Debug);
            logging.AddFilter("Aspire.", LogLevel.Debug);
            // To output logs to the xUnit.net ITestOutputHelper, consider adding a package from https://www.nuget.org/packages?q=xunit+logging
        });

        // TODO: inject parameter, secrets
        app = await appHost.BuildAsync(cts.Token);
        await app!.StartAsync(cts.Token);

        await app!.ResourceNotifications.WaitForResourceHealthyAsync("ui", cts.Token);
        Target = new(app!.GetEndpoint("ui", "https"));
    }

    public async Task DisposeAsync()
    {
        if (app is null)
        {
            return;
        }
        await app!.DisposeAsync();
    }
}
