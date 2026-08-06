using Aspire.Hosting;
using Aspire.Hosting.Testing;
using Microsoft.Extensions.Logging;
using Xunit;

namespace e2e.Fixtures;

[CollectionDefinition(nameof(AppFixture))]
public sealed class AppCollection : ICollectionFixture<AppFixture>;

public sealed class AppFixture : IAsyncLifetime
{
    public AppTestTarget Target { get; private set; } = null!;

    private DistributedApplication? app;

    private Uri ValidUri(string inputUri)
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
            Target = new(UIBaseUri: ValidUri(uiBaseUrl), DemoBaseUri: ValidUri(demoBaseUrl));
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
            "Parameters__auth-jwt-public-key",
            "1RwJkXgznrTP267FaSdzXqeC7wxIL10EcmVmAL2qSeU="
        );
        Environment.SetEnvironmentVariable(
            "Parameters__auth-jwt-private-key",
            "9y7KNcPRFzIlYFVe4jJu7BFJbdD4b+pkqY4ppx0vwk3VHAmReDOetM/brsVpJ3Nep4LvDEgvXQRyZWYAvapJ5Q=="
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

        app = await appHost.BuildAsync(cts.Token);
        await app!.StartAsync(cts.Token);

        await app!.ResourceNotifications.WaitForResourceHealthyAsync("ui", cts.Token);
        Target = new(UIBaseUri: app!.GetEndpoint("ui", "https"), DemoBaseUri: app!.GetEndpoint("demo", "http"));
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
