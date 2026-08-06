using Xunit;

using e2e.Fixtures;

namespace e2e.Tests;

[Collection(nameof(AppFixture))]
public class IntegrationTest(AppFixture fixture)
{
    private static readonly TimeSpan DefaultTimeout = TimeSpan.FromSeconds(30);

    [Fact]
    public async Task GetUIResourceRootReturnsOkStatusCode()
    {
        // Arrange
        using var cts = new CancellationTokenSource(DefaultTimeout);

        // Act
        using var httpClient = new HttpClient()
        {
            BaseAddress = fixture.Target.UIBaseUri
        };
        using var response = await httpClient.GetAsync("/", cts.Token);

        // Assert
        Assert.Equal(HttpStatusCode.OK, response.StatusCode);
    }

    [Fact]
    public async Task GetDemoAppReturnsOkStatusCode()
    {
        // Arrange
        using var cts = new CancellationTokenSource(DefaultTimeout);

        // Act
        using var httpClient = new HttpClient() {
            BaseAddress = fixture.Target.DemoBaseUri
        };
        using var response = await httpClient.GetAsync("/", cts.Token);

        // Assert
        Assert.Equal(HttpStatusCode.OK, response.StatusCode);
    }
}
