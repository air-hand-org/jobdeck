using System.Collections.Concurrent;
using Microsoft.Playwright;

using e2e.Attributes;
using e2e.Fixtures;

namespace e2e.Tests;

[WithTestName]
[Collection(nameof(AppFixture))]
public sealed class WasmBrowserTest(AppFixture fixture) : BrowserE2ETest(fixture)
{
    // Released UI images remain the default target; explicitly opt in for a compatible source/image.
    public static bool IsWasmTarget => string.Equals(
        Environment.GetEnvironmentVariable("E2E_UI_WASM"), "true", StringComparison.OrdinalIgnoreCase);

    private const string JobRpc = "/jobdeck.job.v1.JobService/";
    private const string ExecutionRpc = "/jobdeck.job.v1.ExecutionService/";
    private const string TokenRpc = "/jobdeck.bff.v1.AntiforgeryService/GetAntiforgeryToken";

    [Fact(Skip = "Requires an explicitly selected WASM/gRPC-Web UI target (E2E_UI_WASM=true).", SkipUnless = nameof(IsWasmTarget))]
    public async Task HydratedJobsCrudPollingAndReloginUseFreshAntiforgery()
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
        await page.GetByLabel("Request URL:").FillAsync(new Uri(Fixture.Target.DemoBaseUriFromContainerNetwork, "/200").AbsoluteUri);
        await page.GetByLabel("Request Method:").SelectOptionAsync(new SelectOptionValue { Label = "Get" });
        await page.GetByLabel("Timeout(sec):").FillAsync("30");
        await page.GetByRole(AriaRole.Button, new() { Name = "Create", Exact = true }).ClickAsync();
        var row = page.GetByRole(AriaRole.Row).Filter(new() { HasText = name });
        await Expect(row).ToBeVisibleAsync();
        var editPath = await row.GetByRole(AriaRole.Link, new() { Name = "Edit", Exact = true }).GetAttributeAsync("href");
        Assert.NotNull(editPath);
        var jobPath = editPath[..^"/edit".Length];

        try
        {
            // The authenticated HTML must contain the read model even without JavaScript.
            await using (var ssr = await Browser.NewContextAsync(new() { JavaScriptEnabled = false, StorageState = await Context.StorageStateAsync() }))
            {
                var htmlPage = await ssr.NewPageAsync();
                await htmlPage.GotoAsync(new Uri(Fixture.Target.UIBaseUri, editPath).AbsoluteUri);
                await Expect(htmlPage.GetByLabel("Name:", new() { Exact = true })).ToHaveValueAsync(name);
                await Expect(htmlPage.GetByRole(AriaRole.Button, new() { Name = "Update", Exact = true })).ToBeDisabledAsync();
            }

            var readsBefore = Count(requests, JobRpc + "GetJob");
            await page.GotoAsync(new Uri(Fixture.Target.UIBaseUri, editPath).AbsoluteUri);
            await Expect(page.GetByRole(AriaRole.Button, new() { Name = "Update", Exact = true })).ToBeEnabledAsync();
            Assert.Equal(readsBefore, Count(requests, JobRpc + "GetJob"));
            await UpdateAsync(page, name + "-updated");
            var oldToken = await LastUpdateTokenAsync(requests);

            await page.GotoAsync(new Uri(Fixture.Target.UIBaseUri, jobPath + "/executions").AbsoluteUri);
            var run = page.GetByRole(AriaRole.Button, new() { Name = "Request to run the job" });
            await Expect(run).ToBeEnabledAsync();
            // Two periodic reads prove hydration starts the loop; a submit-triggered read is not enough.
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
            var tokenReads = Count(requests, TokenRpc);
            await LoginAsync(page);
            await page.GotoAsync(new Uri(Fixture.Target.UIBaseUri, editPath).AbsoluteUri);
            await Expect(page.GetByRole(AriaRole.Button, new() { Name = "Update", Exact = true })).ToBeEnabledAsync();
            await UpdateAsync(page, name + "-relogin");
            Assert.True(Count(requests, TokenRpc) > tokenReads, "Relogin must acquire an antiforgery token again.");
            Assert.NotEqual(oldToken, await LastUpdateTokenAsync(requests));

            Assert.Contains(requests, r => new Uri(r.Url).AbsolutePath.StartsWith("/_framework/", StringComparison.Ordinal)
                && new Uri(r.Url).AbsolutePath.EndsWith(".wasm", StringComparison.Ordinal));
            Assert.DoesNotContain(requests, r => new Uri(r.Url).AbsolutePath.StartsWith("/_blazor", StringComparison.Ordinal));
            await Expect(page.Locator("#blazor-error-ui")).Not.ToBeVisibleAsync();
            Assert.Empty(errors);
        }
        finally
        {
            await page.GotoAsync(new Uri(Fixture.Target.UIBaseUri, jobPath + "/delete").AbsoluteUri);
            var delete = page.GetByRole(AriaRole.Button, new() { Name = "Delete", Exact = true });
            await Expect(delete).ToBeEnabledAsync();
            await delete.ClickAsync();
            await Expect(page.GetByRole(AriaRole.Heading, new() { Name = "Jobs list" })).ToBeVisibleAsync();
            await Expect(page.GetByRole(AriaRole.Row).Filter(new() { HasText = name })).ToHaveCountAsync(0);
        }
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
        // Observe the application's own refresh instead of starting a competing reload.
        var refreshed = await page.RunAndWaitForResponseAsync(
            () => page.GetByRole(AriaRole.Button, new() { Name = "Update", Exact = true }).ClickAsync(),
            response => response.Request.Method == "GET" && IsPath(response.Request, editPath));
        await refreshed.FinishedAsync();
        Assert.Contains(name, await refreshed.TextAsync());
        await Expect(page.GetByLabel("Name:", new() { Exact = true })).ToHaveValueAsync(name);
    }

    private static bool IsPath(IRequest request, string path) => new Uri(request.Url).AbsolutePath == path;
    private static int Count(IEnumerable<IRequest> requests, string path) => requests.Count(r => IsPath(r, path));

    private static async Task<string> LastUpdateTokenAsync(IEnumerable<IRequest> requests)
    {
        var token = await requests.Last(r => IsPath(r, JobRpc + "UpdateJob")).HeaderValueAsync("x-csrf-token");
        Assert.False(string.IsNullOrEmpty(token), "The browser mutation must carry an antiforgery header.");
        return token;
    }
}
