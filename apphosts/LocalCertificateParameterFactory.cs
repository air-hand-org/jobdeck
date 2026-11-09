using System.Security.Cryptography;
using System.Security.Cryptography.X509Certificates;

namespace apphosts;

public sealed class LocalCertificateParameterFactory(
    IDistributedApplicationBuilder builder
    , string password
    , string? certDir = null
)
{
    public IResourceBuilder<ParameterResource> AddParameterAsBase64(
        string name, X509KeyUsageFlags keyUsage
    )
    {
        return builder.AddParameter(
            name,
            () => Convert.ToBase64String(LoadOrCreate(name, keyUsage)),
            secret: true
        );
    }

    private byte[] LoadOrCreate(
        string name, X509KeyUsageFlags keyUsage
    )
    {
        var path = Path.Combine(certDir ?? builder.AppHostDirectory, "certs", $"{name}.pfx");

        byte[] pfx;

        try
        {
            Directory.CreateDirectory(Path.GetDirectoryName(path));
            pfx = File.ReadAllBytes(path);
        }
        catch (FileNotFoundException)
        {
            pfx = Create(name, keyUsage);
            using var file = new FileStream(
                path,
                FileMode.CreateNew,
                FileAccess.Write
            );
            file.Write(pfx);
        }
        using var cert = X509CertificateLoader.LoadPkcs12(
            pfx,
            password,
            X509KeyStorageFlags.EphemeralKeySet | X509KeyStorageFlags.Exportable
        );
        return cert.Export(X509ContentType.Pfx, string.Empty);
    }

    private byte[] Create(
        string name, X509KeyUsageFlags keyUsage
    )
    {
        using var rsa = RSA.Create(3072);
        var request = new CertificateRequest(
            new X500DistinguishedName($"CN={name}"),
            rsa,
            HashAlgorithmName.SHA256,
            RSASignaturePadding.Pkcs1
        );

        request.CertificateExtensions.Add(
            new X509BasicConstraintsExtension(
                certificateAuthority: false,
                hasPathLengthConstraint: false,
                pathLengthConstraint: 0,
                critical: true
            )
        );
        request.CertificateExtensions.Add(
            new X509KeyUsageExtension(
                keyUsage,
                critical: true
            )
        );

        var now = DateTimeOffset.UtcNow;

        using var certificate = request.CreateSelfSigned(
            notBefore: now.AddMinutes(-5),
            notAfter: now.AddYears(10)
        );

        return certificate.Export(X509ContentType.Pfx, password);
    }
}
