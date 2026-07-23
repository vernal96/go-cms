# Core seeds

Each subdirectory is an independent versioned seed source exposed through
`seeds.Provider`.

- `shared` is tagged with both `dev` and `prod` and creates the `admin` and
  `manager` groups. It never creates production users.
- `dev` creates public development sites and the `admin` user in the `admin`
  group. Its development-only password is `admin-dev-only-2026`.

Use `console seeds up -tags=dev` for a development database and
`console seeds up -tags=prod` for production group initialization.
