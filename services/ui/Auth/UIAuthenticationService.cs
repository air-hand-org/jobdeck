using System.Security.Claims;
using Microsoft.AspNetCore.Authentication;
using Microsoft.AspNetCore.Authentication.Cookies;
using ui.Client;
using ui.Models;
using ui.Session;

namespace ui.Auth;

public sealed class UIAuthenticationService(AuthApiClient authApi, IUISessionStore sessions)
{
    public async Task<ApiResult> LoginAsync(LoginFormModel model, HttpContext httpContext, CancellationToken cancellationToken)
    {
        var result = await authApi.AuthenticateLocalClientAsync(model, cancellationToken);
        if (!result.IsSuccess || result.Value is null)
        {
            return ApiResult.Failure(result.Error ?? "Login failed");
        }

        var token = result.Value;
        var session = new UISession(
          Guid.NewGuid(),
          token.AccessToken,
          DateTimeOffset.UtcNow.AddSeconds(token.ExpiresInSeconds)
        );
        await sessions.SetSessionAsync(session, cancellationToken);

        var principal = new ClaimsPrincipal(
          new ClaimsIdentity(
              [new Claim("sid", session.SessionId.ToString())],
              CookieAuthenticationDefaults.AuthenticationScheme
          )
        );
        var properties = new AuthenticationProperties
        {
            ExpiresUtc = session.AccessTokenExpiresAt,
            IsPersistent = false,
            AllowRefresh = false,
        };
        await httpContext.SignInAsync(
          CookieAuthenticationDefaults.AuthenticationScheme,
          principal,
          properties
        );

        return ApiResult.Success();
    }

    public async Task LogoutAsync(
        HttpContext httpContext,
        CancellationToken cancellationToken
    )
    {
        var sid = httpContext.User.FindFirstValue("sid");
        if (Guid.TryParse(sid, out var sessionId))
        {
            await sessions.RemoveSessionAsync(sessionId, cancellationToken);
        }

        await httpContext.SignOutAsync(
            CookieAuthenticationDefaults.AuthenticationScheme
        );
    }
}
