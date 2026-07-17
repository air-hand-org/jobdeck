namespace ui.Session;

public sealed record Session(
    Guid SessionId,
    string AccessToken,
    // TODO
    // string ActorId,
    // string DisplayName,
    DateTimeOffset AccessTokenExpiresAt
);
