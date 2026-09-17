# Project filesystem declarations

Each package declares one physical disk and chooses which settings come from Go
and which come from environment configuration. The application registers factories
explicitly in `internal/config.Config.Application`. There is no environment-based
list of disks or automatic registration.

The bundled `public` and `private` declarations fix their codes, Russian labels,
local driver, and visibility in Go. Their `Config` types expose only root, base URL,
and (for the private disk) signing key. Nested `Files.Public` / `Files.Private`
configuration produces `FILES_PUBLIC_*` / `FILES_PRIVATE_*` variables. Process
environment takes precedence over `.env`; required values fail configuration
loading, and driver settings are validated when the manager opens each disk.

A constructor can also be called entirely from Go:

```go
publicfiles.NewFactory(publicfiles.Config{
    Root: "var/media",
    BaseURL: "https://cms.example.org",
})
```

To add an S3 disk, create a package such as `internal/filesystems/media`:

```go
package media

import (
    "github.com/vernal96/go-cms/internal/connectors/corefiles"
    "github.com/vernal96/go-cms-kernel/filesystem"
)

const Code filesystem.Code = "media"

type Config struct {
    Bucket string `envconfig:"BUCKET" required:"true"`
    AccessKeyID string `envconfig:"ACCESS_KEY_ID" required:"true"`
    SecretAccessKey string `envconfig:"SECRET_ACCESS_KEY" required:"true"`
}

func NewFactory(c Config) filesystem.Factory {
    return corefiles.NewFactory(corefiles.Config{
        Code: Code,
        Label: "Медиатека",
        Driver: "s3",
        Visibility: filesystem.VisibilityPrivate,
        S3: corefiles.S3Config{
            Region: "eu-central-1",
            Bucket: c.Bucket,
            AccessKeyID: c.AccessKeyID,
            SecretAccessKey: c.SecretAccessKey,
        },
    })
}
```

Add `Media media.Config` with `envconfig:"MEDIA"` to `FilesConfig`, then register
`media.NewFactory(c.Files.Media)` in `Application.Filesystems`. This exposes
`FILES_MEDIA_BUCKET`, `FILES_MEDIA_ACCESS_KEY_ID`, and
`FILES_MEDIA_SECRET_ACCESS_KEY`. The declaration can instead fix the bucket in Go
or expose region/endpoint as environment parameters. Public S3 disks can configure
`PublicBaseURL`; S3-compatible endpoints use `Endpoint` and `UsePathStyle`.
The existing S3 connector also supports the AWS credential provider chain when
explicit credentials are omitted by a declaration.

Codes identify physical storage in persisted file references; labels are display
metadata. Keep `FILES_INTERNAL_STORAGE`, `FILES_AVATAR_STORAGE`,
`CACHE_FILESYSTEM_STORAGE`, `MAIL_UPLOAD_STORAGE`, profile aliases and file-field
storage restrictions pointing to declared codes. The application manager owns
opening, validation, metadata and closing; declarations do not open disks or read
environment variables during requests.

For an existing local checkout, replace its old `FILES_DISKS` setting with the
individual variables in `.env.example`, retaining its paths, URLs and signing key.
Do not reset files or databases. To retain a different physical driver, express it
in the corresponding Go declaration. The old JSON setting is no longer read.
