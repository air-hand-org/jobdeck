using Microsoft.Extensions.Caching.Memory;

namespace ui.Session;

public sealed class MemoryUISessionStore(IMemoryCache cache) : IUISessionStore
{
    private static string Key(Guid id) => $"session:{id}";

    public Task<UISession?> GetSessionAsync(Guid sessionId, CancellationToken cancellationToken = default)
    {
        cache.TryGetValue(Key(sessionId), out UISession? session);
        return Task.FromResult(session);
    }

    public Task SetSessionAsync(UISession session, CancellationToken cancellationToken = default)
    {
        cache.Set(
            Key(session.SessionId),
            session,
            new MemoryCacheEntryOptions
            {
                SlidingExpiration = TimeSpan.FromHours(8),
                AbsoluteExpirationRelativeToNow = TimeSpan.FromDays(1),
            }
        );
        return Task.CompletedTask;
    }

    public Task RemoveSessionAsync(Guid sessionId, CancellationToken cancellationToken = default)
    {
        cache.Remove(Key(sessionId));
        return Task.CompletedTask;
    }
}
