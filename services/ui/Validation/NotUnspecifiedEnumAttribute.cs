using System.ComponentModel.DataAnnotations;

namespace ui.Validation;

[AttributeUsage(AttributeTargets.Property)]
public sealed class NotUnspecifiedEnumAttribute : ValidationAttribute
{
    public override bool IsValid(object? value)
    {
        if (value is null)
        {
            return false;
        }
        if (!value.GetType().IsEnum)
        {
            return false;
        }
        return Convert.ToInt32(value) != 0; // proto enum unspecified is 0
    }
}
