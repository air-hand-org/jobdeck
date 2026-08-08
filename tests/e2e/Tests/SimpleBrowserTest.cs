using System.Text.RegularExpressions;
using Microsoft.Playwright;
using Microsoft.Playwright.Xunit;

using e2e.Attributes;
using e2e.Fixtures;

namespace e2e.Tests;

[WithTestName]
[Collection(nameof(AppFixture))]
public sealed class SimpleBrowserTest(AppFixture fixture) : BrowserTest
{
    private IBrowserContext context = null!;

    public override async Task InitializeAsync()
    {
        await base.InitializeAsync();
        context = await Browser.NewContextAsync();
        await context.Tracing.StartAsync(new()
        {
            Title = $"{WithTestNameAttribute.CurrentClassName}.{WithTestNameAttribute.CurrentTestName}",
            Screenshots = true,
            Snapshots = true,
            Sources = true
        });
    }

    public override async Task DisposeAsync()
    {
        await context.Tracing.StopAsync(new()
        {
            Path = Path.Combine(
                    Environment.CurrentDirectory,
                    "playwright-traces",
                    $"{WithTestNameAttribute.CurrentClassName}.{WithTestNameAttribute.CurrentTestName}.zip"
                )
        });
        await base.DisposeAsync();
    }

    [Fact]
    public async Task HasLoginLinkMenu()
    {
        var page = await context.NewPageAsync();
        var baseUri = fixture.Target.UIBaseUri;
        await page.GotoAsync(baseUri.AbsoluteUri);

        await Expect(page.GetByRole(AriaRole.Link, new() { Name = "Login" })).ToBeVisibleAsync();
    }

    [Fact]
    public async Task ToBeRedirectedWithoutLogin()
    {
        var page = await context.NewPageAsync();
        var baseUri = fixture.Target.UIBaseUri;
        await page.GotoAsync(baseUri.AbsoluteUri);
        await page.GetByRole(AriaRole.Link, new() { Name = "Jobs" }).ClickAsync();

        await Expect(page).ToHaveURLAsync(
            new Regex(@"/account/login.*$")
        );
    }

    [Fact]
    public async Task CanGotoJobsPageWithLogin()
    {
        var page = await context.NewPageAsync();
        var baseUri = fixture.Target.UIBaseUri;
        await page.GotoAsync(baseUri.AbsoluteUri);
        await page.GetByRole(AriaRole.Link, new() { Name = "Login" }).ClickAsync();

        await Expect(page).ToHaveURLAsync(new Regex(@"/account/login"));

        await page.GetByLabel("Id:").FillAsync("test-user");

        var testUserSecret = Environment.GetEnvironmentVariable("E2E_TEST_USER_SECRET");
        Assert.NotNull(testUserSecret);
        Assert.NotEmpty(testUserSecret);

        await page.GetByLabel("Password:").FillAsync(testUserSecret);
        await page.GetByRole(AriaRole.Button, new() { Name = "Login" }).ClickAsync();

        await Expect(page.GetByRole(AriaRole.Link, new() { Name = "Logout" })).ToBeVisibleAsync();

        await page.GetByRole(AriaRole.Link, new() { Name = "Jobs" }).ClickAsync();

        await Expect(page).ToHaveURLAsync(
            new Regex(@"/jobs$")
        );
        await Expect(page.GetByRole(AriaRole.Heading, new() { Name = "Jobs list" })).ToBeVisibleAsync();
    }
}
