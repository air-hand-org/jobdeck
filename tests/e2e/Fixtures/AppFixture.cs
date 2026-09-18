using Aspire.Hosting;
using Aspire.Hosting.ApplicationModel;
using Aspire.Hosting.Testing;
using Microsoft.Extensions.Logging;
using Xunit;

namespace e2e.Fixtures;

[CollectionDefinition(nameof(AppFixture))]
public sealed class AppCollection : ICollectionFixture<AppFixture>;

public sealed class AppFixture : IAsyncLifetime
{
    private static readonly string[] ResourceLogNames =
    [
        "auth",
        "auth-tls",
        "demo",
        "demo-backend",
        "forward-proxy",
        "job-api",
        "job-api-tls",
        "job-worker",
        "postgres",
        "pubsub",
        "ui",
    ];

    public AppTestTarget Target { get; private set; } = null!;

    private DistributedApplication? app;

    private static Uri ValidUri(string inputUri)
    {
        if (!Uri.TryCreate(inputUri, UriKind.Absolute, out var uri)
            || (uri.Scheme is not "http" and not "https")
        )
        {
            throw new InvalidOperationException("Input uri must be an absolute HTTP(S) URL.");
        }
        return uri;
    }

    public async Task InitializeAsync()
    {
        var uiBaseUrl = Environment.GetEnvironmentVariable("E2E_UI_BASE_URL");
        var demoBaseUrl = Environment.GetEnvironmentVariable("E2E_DEMO_BASE_URL");
        if (!string.IsNullOrWhiteSpace(uiBaseUrl) && !string.IsNullOrWhiteSpace(demoBaseUrl))
        {
            var demoUri = ValidUri(demoBaseUrl);
            Target = new(UIBaseUri: ValidUri(uiBaseUrl), DemoBaseUri: demoUri, DemoBaseUriFromContainerNetwork: demoUri);
            return;
        }

        using var cts = new CancellationTokenSource(TimeSpan.FromMinutes(5));

        Environment.SetEnvironmentVariable(
            "API_AUTH_MODE", "REQUIRED" // disable fallback
        );
        // MEMO: hard coded secrets to test environment
        Environment.SetEnvironmentVariable(
            "Parameters__test-user-client-secret-hash",
            "$argon2id$v=19$m=19456,t=2,p=1$eppSy5LBhs7851suliZtBg$vbdeXijopLCHjMuCtb85FPjkNvJK9dTBiWGd1T+gJcc"
        );
        Environment.SetEnvironmentVariable(
            "E2E_TEST_USER_SECRET",
            "test-e2e"
        );

        Environment.SetEnvironmentVariable(
            "Parameters__job-worker-client-secret",
            "test-worker"
        );
        Environment.SetEnvironmentVariable(
            "Parameters__job-worker-client-secret-hash",
            "$argon2id$v=19$m=19456,t=2,p=1$uuhh20q0GZt7RTIT2TttQw$WoXQCPm3wGZt89DsiyUfueugcmG1DbrdlSk4m/R8qKw"
        );

        Environment.SetEnvironmentVariable(
            "Parameters__auth-jwt-private-key",
            "9y7KNcPRFzIlYFVe4jJu7BFJbdD4b+pkqY4ppx0vwk3VHAmReDOetM/brsVpJ3Nep4LvDEgvXQRyZWYAvapJ5Q=="
        );
        Environment.SetEnvironmentVariable(
            "Parameters__auth-jwt-key-id",
            "jobdeck-auth-e2e-1"
        );

        var appHost = await DistributedApplicationTestingBuilder.CreateAsync<Projects.Apphosts>(
            [
                "UseVolumes=false",
            ],
            cts.Token
        );
        appHost.Services.AddLogging(logging =>
        {
            logging.SetMinimumLevel(LogLevel.Information);
            // Override the logging filters from the app's configuration
            logging.AddFilter(appHost.Environment.ApplicationName, LogLevel.Debug);
            logging.AddFilter("Aspire.", LogLevel.Information);
            // To output logs to the xUnit.net ITestOutputHelper, consider adding a package from https://www.nuget.org/packages?q=xunit+logging
        });

        try
        {
            app = await appHost.BuildAsync(cts.Token);
            await app.StartAsync(cts.Token);

            await app.ResourceNotifications.WaitForResourceHealthyAsync("ui", cts.Token);
            Target = new(
                UIBaseUri: app.GetEndpoint("ui", "https"),
                DemoBaseUri: app.GetEndpoint("demo", "https"),
                DemoBaseUriFromContainerNetwork: app.GetEndpointForNetwork(
                    "demo",
                    KnownNetworkIdentifiers.DefaultAspireContainerNetwork,
                    "https"
                )
            );
        }
        catch
        {
            await DisposeAsync();
            throw;
        }
    }

    public async Task DisposeAsync()
    {
        if (app is null)
        {
            return;
        }

        try
        {
            await CaptureResourceLogsAsync();
        }
        finally
        {
            await app.DisposeAsync();
        }
    }

    private async Task CaptureResourceLogsAsync()
    {
        var logDirectory = Path.Combine("TestResults", "aspire-logs");
        Directory.CreateDirectory(logDirectory);

        var resourceLogs = app!.Services.GetRequiredService<ResourceLoggerService>();
        var appModel = app.Services.GetRequiredService<DistributedApplicationModel>();

        await Task.WhenAll(appModel.Resources
            .Where(resource => ResourceLogNames.Contains(resource.Name, StringComparer.Ordinal))
            .Select(async resource =>
        {
            var logPath = Path.Combine(logDirectory, $"{resource.Name}.log");
            await using var writer = new StreamWriter(logPath);

            await foreach (var batch in resourceLogs.GetAllAsync(resource))
            {
                foreach (var entry in batch)
                {
                    await writer.WriteLineAsync(entry.Content);
                }
            }
        }));
    }
}
