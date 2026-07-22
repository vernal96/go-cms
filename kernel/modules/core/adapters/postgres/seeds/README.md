# Core seeds

Core currently has no mandatory data shared by every application instance.
Add paired `*.up.sql` and `*.down.sql` files here when such data appears, then
expose the directory through `seeds.Provider` on the PostgreSQL database adapter.
