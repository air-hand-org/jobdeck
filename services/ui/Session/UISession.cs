namespace ui.Session;

public sealed record UISession(
    Guid SessionId,
    string AccessToken,
    // TODO
    // string ActorId,
    // string DisplayName,
    DateTimeOffset AccessTokenExpiresAt
);
