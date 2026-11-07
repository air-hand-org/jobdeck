using System.Reflection;
using Xunit.v3;

namespace e2e.Attributes;

public sealed class WithTestNameAttribute : BeforeAfterTestAttribute
{
    public static string CurrentTestName = string.Empty;
    public static string CurrentClassName = string.Empty;

    public override void Before(MethodInfo methodInfo, IXunitTest test)
    {
        CurrentTestName = methodInfo.Name;
        CurrentClassName = methodInfo.DeclaringType!.Name;
    }

    public override void After(MethodInfo methodInfo, IXunitTest test) { }
}
