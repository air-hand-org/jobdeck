using System.Security.Claims;
using Microsoft.AspNetCore.Authentication;
using Microsoft.AspNetCore.Authentication.Cookies;
using ui.Session;

namespace ui.Auth;

public sealed class UICookieAuthenticationEvents(IUISessionStore sessions) : CookieAuthenticationEvents
{
    public override async Task ValidatePrincipal(CookieValidatePrincipalContext context)
    {
        var sessionIdValue = context.Principal?.FindFirstValue("sid");
        if (!Guid.TryParse(sessionIdValue, out var sessionId))
        {
            context.RejectPrincipal();
            await context.HttpContext.SignOutAsync(context.Scheme.Name);
            return;
        }

        var session = await sessions.GetSessionAsync(sessionId, context.HttpContext.RequestAborted);
        if (session is null)
        {
            context.RejectPrincipal();
            await context.HttpContext.SignOutAsync(context.Scheme.Name);
            return;
        }

        await base.ValidatePrincipal(context);
    }
}
