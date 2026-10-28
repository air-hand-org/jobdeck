using Aspire.Hosting;

namespace e2e.Fixtures;

internal static class AppStartup
{
    internal static async Task RunAsync(
        ResourceNotificationService notifications,
        Func<CancellationToken, Task> startAndWait,
        CancellationToken cancellationToken)
    {
        using var startup = CancellationTokenSource.CreateLinkedTokenSource(cancellationToken);
        using var monitoring = CancellationTokenSource.CreateLinkedTokenSource(cancellationToken);
        ResourceEvent? failed = null;
        var monitor = MonitorAsync();
        try
        {
            await startAndWait(startup.Token);
            if (failed is not null)
            {
                throw new DistributedApplicationException($"Resource '{failed.ResourceId}' entered {KnownResourceStates.FailedToStart} during startup.");
            }
        }
        catch (OperationCanceledException error) when (failed is not null && !cancellationToken.IsCancellationRequested)
        {
            throw new DistributedApplicationException(
                $"Resource '{failed.ResourceId}' entered {KnownResourceStates.FailedToStart} during startup.", error);
        }
        finally
        {
            await monitoring.CancelAsync();
            try
            {
                await monitor;
            }
            catch (OperationCanceledException) when (monitoring.IsCancellationRequested)
            {
            }
        }

        async Task MonitorAsync()
        {
            await foreach (var update in notifications.WatchAsync(monitoring.Token))
            {
                // Finished/Exited are normal for successful one-shot migrators.
                if (update.Snapshot.State?.Text == KnownResourceStates.FailedToStart)
                {
                    failed = update;
                    await startup.CancelAsync();
                    return;
                }
            }
        }
    }
}
