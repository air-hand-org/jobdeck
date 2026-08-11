using Microsoft.Playwright;

using e2e.Attributes;
using e2e.Fixtures;

namespace e2e.Tests;

[WithTestName]
[Collection(nameof(AppFixture))]
public sealed class ForwardProxyBrowserTest(AppFixture fixture) : BrowserE2ETest(fixture)
{
    private static Uri ToForwardProxyDemoUri(Uri demoBaseUri)
    {
        if (!demoBaseUri.IsLoopback)
        {
            return demoBaseUri;
        }

        var builder = new UriBuilder(demoBaseUri)
        {
            Host = "host.docker.internal",
        };
        return builder.Uri;
    }

    private async Task<IPage> RequestExecutionAsync(Uri requestUri)
    {
        var page = await Context.NewPageAsync();
        await page.GotoAsync(Fixture.Target.UIBaseUri.AbsoluteUri);
        await page.GetByRole(AriaRole.Link, new() { Name = "Login" }).ClickAsync();

        var testUserSecret = Environment.GetEnvironmentVariable("E2E_TEST_USER_SECRET");
        Assert.NotNull(testUserSecret);
        Assert.NotEmpty(testUserSecret);

        await page.GetByLabel("Id:").FillAsync("test-user");
        await page.GetByLabel("Password:").FillAsync(testUserSecret);
        await page.GetByRole(AriaRole.Button, new() { Name = "Login" }).ClickAsync();

        await page.GetByRole(AriaRole.Link, new() { Name = "Jobs" }).ClickAsync();
        await page.GetByRole(AriaRole.Link, new() { Name = "Create New" }).ClickAsync();

        var createButton = page.GetByRole(AriaRole.Button, new() { Name = "Create" });
        await Expect(createButton).ToBeEnabledAsync();

        var jobName = $"e2e-forward-proxy-{Guid.NewGuid():N}";
        await page.GetByLabel("Name:").FillAsync(jobName);
        await page.GetByLabel("Request URL:").FillAsync(requestUri.AbsoluteUri);
        await page.Locator("#method").SelectOptionAsync("Get");
        await page.GetByLabel("Timeout(sec):").FillAsync("5");
        await createButton.ClickAsync();

        var jobRow = page.Locator("tr").Filter(new() { HasText = jobName });
        await Expect(jobRow).ToBeVisibleAsync();
        await jobRow.GetByRole(AriaRole.Link, new() { Name = "Executions" }).ClickAsync();
        await page.GetByRole(AriaRole.Button, new() { Name = "Request to run the job" }).ClickAsync();

        return page;
    }

    [Fact]
    public async Task CanExecuteAllowedDemoUrlThroughForwardProxy()
    {
        var demoUri = new Uri(Fixture.Target.DemoBaseUri, "/200");
        var page = await RequestExecutionAsync(ToForwardProxyDemoUri(demoUri));

        await Expect(page.GetByText("Succeeded", new() { Exact = true }))
            .ToBeVisibleAsync(new() { Timeout = 30_000 });
    }

    [Fact]
    public async Task CanExecuteAllowedHttpsUrlThroughForwardProxy()
    {
        var page = await RequestExecutionAsync(new Uri("https://www.google.com/"));

        await Expect(page.GetByText("Succeeded", new() { Exact = true }))
            .ToBeVisibleAsync(new() { Timeout = 30_000 });
    }

    [Fact]
    public async Task CannotExecuteForbiddenLoopbackUrlThroughForwardProxy()
    {
        var page = await RequestExecutionAsync(new Uri("http://127.0.0.1/"));

        await Expect(page.GetByText("Failed", new() { Exact = true }))
            .ToBeVisibleAsync(new() { Timeout = 30_000 });
    }
}
