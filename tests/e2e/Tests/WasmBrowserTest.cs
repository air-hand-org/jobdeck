using System.Collections.Concurrent;
using Microsoft.Playwright;

using e2e.Attributes;
using e2e.Fixtures;

namespace e2e.Tests;

[WithTestName]
[Collection(nameof(AppFixture))]
public sealed class WasmBrowserTest(AppFixture fixture) : BrowserE2ETest(fixture)
{
    private const string ExecutionRpc = "/jobdeck.job.v1.ExecutionService/";

    [Fact]
    public async Task JobsCrudExecutionAndReloginPersistChanges()
    {
        var page = await Context.NewPageAsync();
        var requests = new ConcurrentQueue<IRequest>();
        page.Request += (_, request) => requests.Enqueue(request);
        var errors = new ConcurrentQueue<string>();
        page.PageError += (_, error) => errors.Enqueue(error);
        await LoginAsync(page);

        var name = $"wasm-e2e-{Guid.NewGuid():N}";
        await page.GotoAsync(new Uri(Fixture.Target.UIBaseUri, "/jobs/new").AbsoluteUri);
        await Expect(page.GetByRole(AriaRole.Button, new() { Name = "Create", Exact = true })).ToBeEnabledAsync();
        await page.GetByLabel("Name:", new() { Exact = true }).FillAsync(name);
        var requestUrl = new Uri(Fixture.Target.DemoBaseUriFromContainerNetwork, "/200").AbsoluteUri;
        await page.GetByLabel("Request URL:").FillAsync(requestUrl);
        await page.GetByLabel("Request Method:").SelectOptionAsync(new SelectOptionValue { Label = "Get" });
        await page.GetByLabel("Timeout(sec):").FillAsync("30");
        await page.GetByRole(AriaRole.Button, new() { Name = "Create", Exact = true }).ClickAsync();
        var row = page.GetByRole(AriaRole.Row).Filter(new() { HasText = name });
        await Expect(row).ToBeVisibleAsync();
        var editPath = await row.GetByRole(AriaRole.Link, new() { Name = "Edit", Exact = true }).GetAttributeAsync("href");
        Assert.NotNull(editPath);
        var jobPath = editPath[..^"/edit".Length];

        await page.GotoAsync(new Uri(Fixture.Target.UIBaseUri, editPath).AbsoluteUri);
        await Expect(page.GetByLabel("Name:", new() { Exact = true })).ToHaveValueAsync(name);
        await Expect(page.GetByLabel("Request URL:")).ToHaveValueAsync(requestUrl);
        await Expect(page.GetByLabel("Request Method:").Locator("option:checked")).ToHaveTextAsync("Get");
        await Expect(page.GetByLabel("Timeout(sec):")).ToHaveValueAsync("30");
        await UpdateAsync(page, name + "-updated");

        await page.GotoAsync(new Uri(Fixture.Target.UIBaseUri, jobPath + "/executions").AbsoluteUri);
        var run = page.GetByRole(AriaRole.Button, new() { Name = "Request to run the job" });
        await Expect(run).ToBeEnabledAsync();
        // Observe recurring background polling separately from submit-triggered refreshes.
        await page.WaitForRequestAsync(r => IsPath(r, ExecutionRpc + "ListExecutions"), new() { Timeout = 15000 });
        await page.WaitForRequestAsync(r => IsPath(r, ExecutionRpc + "ListExecutions"), new() { Timeout = 15000 });
        await run.ClickAsync();
        await Expect(page.GetByRole(AriaRole.Cell, new() { Name = "Succeeded", Exact = true })).ToBeVisibleAsync(new() { Timeout = 60000 });
        await page.GetByRole(AriaRole.Link, new() { Name = "Back to list" }).ClickAsync();
        await Expect(page.GetByRole(AriaRole.Heading, new() { Name = "Jobs list" })).ToBeVisibleAsync();
        var polls = Count(requests, ExecutionRpc + "ListExecutions");
        // Observe more than one five-second polling period; no session-expiration wait.
        await Task.Delay(TimeSpan.FromSeconds(6), TestContext.Current.CancellationToken);
        Assert.Equal(polls, Count(requests, ExecutionRpc + "ListExecutions"));

        await page.GetByRole(AriaRole.Link, new() { Name = "Logout", Exact = true }).ClickAsync();
        await page.GetByRole(AriaRole.Button, new() { Name = "Logout", Exact = true }).ClickAsync();
        await page.GotoAsync(new Uri(Fixture.Target.UIBaseUri, "/jobs").AbsoluteUri);
        await Expect(page.GetByLabel("Id:")).ToBeVisibleAsync();
        await LoginAsync(page);
        await page.GotoAsync(new Uri(Fixture.Target.UIBaseUri, editPath).AbsoluteUri);
        await Expect(page.GetByLabel("Name:", new() { Exact = true })).ToHaveValueAsync(name + "-updated");
        await Expect(page.GetByRole(AriaRole.Button, new() { Name = "Update", Exact = true })).ToBeEnabledAsync();
        await UpdateAsync(page, name + "-relogin");

        Assert.Contains(requests, r => new Uri(r.Url).AbsolutePath.StartsWith("/_framework/", StringComparison.Ordinal)
            && new Uri(r.Url).AbsolutePath.EndsWith(".wasm", StringComparison.Ordinal));
        Assert.DoesNotContain(requests, r => new Uri(r.Url).AbsolutePath.StartsWith("/_blazor", StringComparison.Ordinal));
        await Expect(page.Locator("#blazor-error-ui")).Not.ToBeVisibleAsync();
        Assert.Empty(errors);
        await page.GotoAsync(new Uri(Fixture.Target.UIBaseUri, jobPath + "/delete").AbsoluteUri);
        var delete = page.GetByRole(AriaRole.Button, new() { Name = "Delete", Exact = true });
        await Expect(delete).ToBeEnabledAsync();
        await delete.ClickAsync();
        await Expect(page.GetByRole(AriaRole.Heading, new() { Name = "Jobs list" })).ToBeVisibleAsync();
        await Expect(page.GetByRole(AriaRole.Row).Filter(new() { HasText = name })).ToHaveCountAsync(0);
        await page.ReloadAsync();
        await Expect(page.GetByRole(AriaRole.Heading, new() { Name = "Jobs list" })).ToBeVisibleAsync();
        await Expect(page.GetByRole(AriaRole.Row).Filter(new() { HasText = name })).ToHaveCountAsync(0);
    }

    private async Task LoginAsync(IPage page)
    {
        await page.GotoAsync(new Uri(Fixture.Target.UIBaseUri, "/account/login").AbsoluteUri);
        await page.GetByLabel("Id:").FillAsync("test-user");
        var secret = Environment.GetEnvironmentVariable("E2E_TEST_USER_SECRET");
        Assert.False(string.IsNullOrEmpty(secret), "E2E_TEST_USER_SECRET must be configured.");
        await page.GetByLabel("Password:").FillAsync(secret);
        await page.GetByRole(AriaRole.Button, new() { Name = "Login", Exact = true }).ClickAsync();
        await Expect(page.GetByRole(AriaRole.Link, new() { Name = "Logout", Exact = true })).ToBeVisibleAsync();
    }

    private async Task UpdateAsync(IPage page, string name)
    {
        await page.GetByLabel("Name:", new() { Exact = true }).FillAsync(name);
        var editPath = new Uri(page.Url).AbsolutePath;
        // Observe the application's refresh without assuming its HTML contains the read model.
        var refreshed = await page.RunAndWaitForResponseAsync(
            () => page.GetByRole(AriaRole.Button, new() { Name = "Update", Exact = true }).ClickAsync(),
            response => response.Request.Method == "GET" && IsPath(response.Request, editPath));
        await refreshed.FinishedAsync();
        await Expect(page.GetByRole(AriaRole.Button, new() { Name = "Update", Exact = true })).ToBeEnabledAsync();
        await page.ReloadAsync();
        await Expect(page.GetByLabel("Name:", new() { Exact = true })).ToHaveValueAsync(name);
    }

    private static bool IsPath(IRequest request, string path) => new Uri(request.Url).AbsolutePath == path;
    private static int Count(IEnumerable<IRequest> requests, string path) => requests.Count(r => IsPath(r, path));

}
