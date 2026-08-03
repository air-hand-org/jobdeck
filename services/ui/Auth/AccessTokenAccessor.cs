using System.Security.Claims;
using Microsoft.AspNetCore.Components.Authorization;
using ui.Session;

namespace ui.Auth;

public sealed class AccessTokenAccessor(AuthenticationStateProvider provider, IUISessionStore sessions)
{
    public async Task<string?> GetAsync(CancellationToken cancellationToken = default)
    {
        var state = await provider.GetAuthenticationStateAsync();
        var sid = state.User.FindFirstValue("sid");
        if (!Guid.TryParse(sid, out var sessionId))
        {
            return null;
        }

        var session = await sessions.GetSessionAsync(sessionId, cancellationToken);
        if (session is null)
        {
            return null;
        }
        if (session.AccessTokenExpiresAt <= DateTimeOffset.UtcNow)
        {
            await sessions.RemoveSessionAsync(sessionId, cancellationToken);
            return null;
        }

        return session.AccessToken;
    }
}
