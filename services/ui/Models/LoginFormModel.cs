using System.ComponentModel.DataAnnotations;

namespace ui.Models;

public sealed class LoginFormModel
{
    [Required]
    public string? Id { get; set; }

    [Required]
    public string? Password { get; set; }
}
