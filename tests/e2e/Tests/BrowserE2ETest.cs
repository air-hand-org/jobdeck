using Microsoft.Playwright;
using Microsoft.Playwright.Xunit;

using e2e.Attributes;
using e2e.Fixtures;

namespace e2e.Tests;

public abstract class BrowserE2ETest(AppFixture fixture) : BrowserTest
{
    protected AppFixture Fixture { get; } = fixture;

    protected IBrowserContext Context { get; private set; } = null!;

    public override async Task InitializeAsync()
    {
        await base.InitializeAsync();
        Context = await Browser.NewContextAsync();
        await Context.Tracing.StartAsync(new()
        {
            Title = $"{WithTestNameAttribute.CurrentClassName}.{WithTestNameAttribute.CurrentTestName}",
            Screenshots = true,
            Snapshots = true,
            Sources = true,
        });
    }

    public override async Task DisposeAsync()
    {
        if (Context is not null)
        {
            await Context.Tracing.StopAsync(new()
            {
                Path = Path.Combine(
                    Environment.CurrentDirectory,
                    "playwright-traces",
                    $"{WithTestNameAttribute.CurrentClassName}.{WithTestNameAttribute.CurrentTestName}.zip"
                ),
            });
            await Context.DisposeAsync();
        }

        await base.DisposeAsync();
    }
}
