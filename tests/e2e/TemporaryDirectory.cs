namespace e2e;

public sealed class TempDirectory(string subdir) : IDisposable
{
    public DirectoryInfo Dir { get; } = Directory.CreateTempSubdirectory(subdir);

    public void Dispose() => Dir.Delete(recursive: true);
}
