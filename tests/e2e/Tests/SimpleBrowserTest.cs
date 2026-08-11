using System.Text.RegularExpressions;
using Microsoft.Playwright;

using e2e.Attributes;
using e2e.Fixtures;

namespace e2e.Tests;

[WithTestName]
[Collection(nameof(AppFixture))]
public sealed partial class SimpleBrowserTest(AppFixture fixture) : BrowserE2ETest(fixture)
{
    [GeneratedRegex(@"/account/login")]
    private static partial Regex AccountLoginUrlRegex();

    [GeneratedRegex(@"/jobs$")]
    private static partial Regex JobsUrlRegex();

    [Fact]
    public async Task HasLoginLinkMenu()
    {
        var page = await Context.NewPageAsync();
        var baseUri = Fixture.Target.UIBaseUri;
        await page.GotoAsync(baseUri.AbsoluteUri);

        await Expect(page.GetByRole(AriaRole.Link, new() { Name = "Login" })).ToBeVisibleAsync();
    }

    [Fact]
    public async Task ToBeRedirectedWithoutLogin()
    {
        var page = await Context.NewPageAsync();
        var baseUri = Fixture.Target.UIBaseUri;
        await page.GotoAsync(baseUri.AbsoluteUri);
        await page.GetByRole(AriaRole.Link, new() { Name = "Jobs" }).ClickAsync();

        await Expect(page).ToHaveURLAsync(
            AccountLoginUrlRegex()
        );
    }

    [Fact]
    public async Task CanGotoJobsPageWithLogin()
    {
        var page = await Context.NewPageAsync();
        var baseUri = Fixture.Target.UIBaseUri;
        await page.GotoAsync(baseUri.AbsoluteUri);
        await page.GetByRole(AriaRole.Link, new() { Name = "Login" }).ClickAsync();

        await Expect(page).ToHaveURLAsync(AccountLoginUrlRegex());

        await page.GetByLabel("Id:").FillAsync("test-user");

        var testUserSecret = Environment.GetEnvironmentVariable("E2E_TEST_USER_SECRET");
        Assert.NotNull(testUserSecret);
        Assert.NotEmpty(testUserSecret);

        await page.GetByLabel("Password:").FillAsync(testUserSecret);
        await page.GetByRole(AriaRole.Button, new() { Name = "Login" }).ClickAsync();

        await Expect(page.GetByRole(AriaRole.Link, new() { Name = "Logout" })).ToBeVisibleAsync();

        await page.GetByRole(AriaRole.Link, new() { Name = "Jobs" }).ClickAsync();

        await Expect(page).ToHaveURLAsync(
            JobsUrlRegex()
        );
        await Expect(page.GetByRole(AriaRole.Heading, new() { Name = "Jobs list" })).ToBeVisibleAsync();
    }
}
