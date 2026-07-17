using System.Diagnostics;

namespace ui.Telemetry;

public static class UiTelemetry
{
  public const string ActivitySourceName = "jobdeck/ui";
  public static readonly ActivitySource Source = new(ActivitySourceName);
}
