namespace ui.Session;

public interface IUISessionStore
{
    Task<UISession?> GetSessionAsync(Guid sessionId, CancellationToken cancellationToken);
    Task SetSessionAsync(UISession session, CancellationToken cancellationToken);
    Task RemoveSessionAsync(Guid sessionId, CancellationToken cancellationToken);
}
