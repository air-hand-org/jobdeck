using System.Diagnostics;
using Aspire.Hosting;
using e2e.Fixtures;
using Microsoft.Extensions.Logging.Abstractions;

namespace e2e.Tests;

public sealed class AppStartupTests
{
    [Fact]
    public async Task FailedToStartInterruptsPendingStartup()
    {
        using var notifications = CreateNotifications();
        using var deadline = new CancellationTokenSource(TimeSpan.FromSeconds(3));
        var resource = new ExecutableResource("broken", "unused", ".");
        var stopped = false;
        var elapsed = Stopwatch.StartNew();

        var error = await Assert.ThrowsAsync<DistributedApplicationException>(() => AppStartup.RunAsync(
            notifications,
            async token =>
            {
                await notifications.PublishUpdateAsync(resource, snapshot => snapshot with
                {
                    State = new ResourceStateSnapshot(KnownResourceStates.FailedToStart, null)
                });
                try
                {
                    await Task.Delay(Timeout.InfiniteTimeSpan, token);
                }
                finally
                {
                    stopped = true;
                }
            },
            deadline.Token));

        Assert.Contains("broken", error.Message);
        Assert.Contains(KnownResourceStates.FailedToStart, error.Message);
        Assert.True(stopped);
        Assert.False(deadline.IsCancellationRequested);
        Assert.True(elapsed.Elapsed < TimeSpan.FromSeconds(2), elapsed.Elapsed.ToString());
    }

    [Theory]
    [InlineData("Finished")]
    [InlineData("Exited")]
    [InlineData("Running")]
    public async Task OrdinaryStatesDoNotInterruptStartup(string state)
    {
        using var notifications = CreateNotifications();
        using var deadline = new CancellationTokenSource(TimeSpan.FromSeconds(3));
        var resource = new ExecutableResource("migrator", "unused", ".");
        await AppStartup.RunAsync(notifications, async token =>
        {
            await notifications.PublishUpdateAsync(resource, snapshot => snapshot with
            {
                State = new ResourceStateSnapshot(state, null)
            });
            await Task.Delay(50, token);
            Assert.False(token.IsCancellationRequested);
        }, deadline.Token);
        // The completed monitor must not cancel a later startup or leak a subscription.
        await notifications.PublishUpdateAsync(resource, snapshot => snapshot with
        {
            State = new ResourceStateSnapshot(KnownResourceStates.FailedToStart, null)
        });
    }

    [Fact]
    public async Task DeadlineStillCancelsPendingStartup()
    {
        using var notifications = CreateNotifications();
        using var deadline = new CancellationTokenSource(TimeSpan.FromMilliseconds(100));
        await Assert.ThrowsAnyAsync<OperationCanceledException>(() => AppStartup.RunAsync(
            notifications, token => Task.Delay(Timeout.InfiniteTimeSpan, token), deadline.Token));
        Assert.True(deadline.IsCancellationRequested);
    }

    [Fact]
    public async Task OriginalStartupExceptionIsPreserved()
    {
        using var notifications = CreateNotifications();
        var original = new InvalidOperationException("original startup error");
        var error = await Assert.ThrowsAsync<InvalidOperationException>(() => AppStartup.RunAsync(
            notifications, _ => Task.FromException(original), CancellationToken.None));
        Assert.Same(original, error);
    }

    [Fact]
    public async Task OriginalStartupExceptionWinsOverConcurrentFailedToStart()
    {
        using var notifications = CreateNotifications();
        var resource = new ExecutableResource("broken", "unused", ".");
        var original = new InvalidOperationException("original launch error");
        using var deadline = new CancellationTokenSource(TimeSpan.FromSeconds(3));
        var error = await Assert.ThrowsAsync<InvalidOperationException>(() => AppStartup.RunAsync(
            notifications, async token =>
            {
                await notifications.PublishUpdateAsync(resource, snapshot => snapshot with
                {
                    State = new ResourceStateSnapshot(KnownResourceStates.FailedToStart, null)
                });
                try
                {
                    await Task.Delay(Timeout.InfiniteTimeSpan, token);
                }
                catch (OperationCanceledException)
                {
                    throw original;
                }
            }, deadline.Token));
        Assert.Same(original, error);
    }

    [Fact]
    public async Task ActualAspireStartupFailsFastForMissingExecutable()
    {
        var builder = await DistributedApplicationTestingBuilder.CreateAsync<Projects.StartupTestAppHost>(TestContext.Current.CancellationToken);
        builder.AddExecutable("broken", "jobdeck-deliberately-missing-executable", ".");
        var waiting = builder.AddExecutable("waiting", "unused", ".");
        // Keep StartAsync pending, as real dependency/environment preparation can do.
        builder.Eventing.Subscribe<BeforeResourceStartedEvent>(waiting.Resource,
            (_, token) => Task.Delay(Timeout.InfiniteTimeSpan, token));
        await using var app = await builder.BuildAsync(TestContext.Current.CancellationToken);
        using var deadline = new CancellationTokenSource(TimeSpan.FromSeconds(30));
        var elapsed = Stopwatch.StartNew();
        var error = await Assert.ThrowsAsync<DistributedApplicationException>(() => AppStartup.RunAsync(
            app.ResourceNotifications, async token =>
            {
                await app.StartAsync(token);
            }, deadline.Token));
        Assert.Contains("broken", error.Message);
        Assert.False(deadline.IsCancellationRequested);
        Assert.True(elapsed.Elapsed < TimeSpan.FromSeconds(15), elapsed.Elapsed.ToString());
    }

    [Fact]
    public async Task ActualOneShotCompletionAndHealthyResourceAreAccepted()
    {
        var builder = await DistributedApplicationTestingBuilder.CreateAsync<Projects.StartupTestAppHost>(TestContext.Current.CancellationToken);
        var migrator = builder.AddExecutable("migrator", "dotnet", ".", "--version");
        builder.AddExecutable("healthy", "node", ".", "-e", "setInterval(() => {}, 1000)")
            .WaitForCompletion(migrator);
        await using var app = await builder.BuildAsync(TestContext.Current.CancellationToken);
        using var deadline = new CancellationTokenSource(TimeSpan.FromSeconds(30));
        await AppStartup.RunAsync(app.ResourceNotifications, async token =>
        {
            await app.StartAsync(token);
            await app.ResourceNotifications.WaitForResourceAsync("migrator", KnownResourceStates.Finished, token);
            await app.ResourceNotifications.WaitForResourceHealthyAsync("healthy", token);
        }, deadline.Token);
        Assert.False(deadline.IsCancellationRequested);
    }

    private static ResourceNotificationService CreateNotifications() => new(
        NullLogger<ResourceNotificationService>.Instance,
        null!,
        new ServiceCollection().BuildServiceProvider(),
        new ResourceLoggerService());
}
