## Codebase Discovery

Use `codebase-memory-mcp` and call `get_architecture` for the current project.

## Repository Architecture

- `backend/` is the only root for the Go backend and its infrastructure files.
- Every frontend is an independent top-level application, such as `frontend-admin/`.
- Never place frontend source code, Node tooling, or frontend build artifacts inside `backend/`.
- Frontends communicate with the backend only through its public HTTP API. Do not share or import source code across application boundaries.
- Do not embed frontend assets into the Go binary or add compatibility paths between applications.

Не делаем легаси проверок и фолбэков. Пишем все как с нуля.
