namespace ui.Session;

public interface ISessionStore
{
    Task<Session?> GetSessionAsync(Guid sessionId, CancellationToken cancellationToken);
    Task SetSessionAsync(Session session, CancellationToken cancellationToken);
    Task RemoveSessionAsync(Guid sessionId, CancellationToken cancellationToken);
}
