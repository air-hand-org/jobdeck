using System.Diagnostics;
using Aspire.Hosting;

namespace apphosts;

internal static class JobWorkerPrebuild
{
    internal const string PackagePath = "./cmd/worker";

    // Independent of publish-mode resource conversion: this warms the host Go
    // cache, not the Docker build cache. It is never a dependency of build/deploy.
#pragma warning disable ASPIREPIPELINES001
    internal static void Register(IDistributedApplicationBuilder builder, string repoRoot)
    {
        builder.Pipeline.AddStep("prebuild-job-worker", async context =>
        {
            var directory = Path.Combine(repoRoot, "services", "job-worker");
            if (!File.Exists(Path.Combine(directory, "go.mod")))
            {
                throw new InvalidOperationException("Prebuilding job-worker requires services/job-worker/go.mod.");
            }

            // Match AddGoApp's default go run compiler settings: inherit the host
            // environment, without Docker's CGO_ENABLED=0 / GOOS overrides.
            var output = Path.Combine(Path.GetTempPath(), $"jobdeck-worker-{Guid.NewGuid():N}");
            var startInfo = new ProcessStartInfo("go")
            {
                WorkingDirectory = directory,
                UseShellExecute = false,
            };
            foreach (var argument in new[] { "build", "-o", output, PackagePath })
            {
                startInfo.ArgumentList.Add(argument);
            }
            try
            {
                using var process = Process.Start(startInfo)
                    ?? throw new InvalidOperationException("Could not start the Go compiler.");
                try
                {
                    await process.WaitForExitAsync(context.CancellationToken);
                }
                catch (OperationCanceledException)
                {
                    process.Kill(entireProcessTree: true);
                    await process.WaitForExitAsync(CancellationToken.None);
                    throw;
                }
                if (process.ExitCode != 0)
                {
                    throw new InvalidOperationException($"job-worker prebuild failed (go build exit code {process.ExitCode}).");
                }
            }
            finally
            {
                File.Delete(output);
            }
        });
    }
#pragma warning restore ASPIREPIPELINES001
}
