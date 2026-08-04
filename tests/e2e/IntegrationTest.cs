using e2e.Fixtures;

namespace e2e;

[Collection(nameof(AppFixture))]
public class IntegrationTest(AppFixture fixture)
{
    private static readonly TimeSpan DefaultTimeout = TimeSpan.FromSeconds(30);

    [Fact]
    public async Task GetUIResourceRootReturnsOkStatusCode()
    {
        // Arrange
        var cancellationToken = CancellationToken.None;

        // Act
        using var httpClient = new HttpClient() {
            BaseAddress = fixture.Target.BaseUri
        };
        using var response = await httpClient.GetAsync("/", cancellationToken);

        // Assert
        Assert.Equal(HttpStatusCode.OK, response.StatusCode);
    }
}
