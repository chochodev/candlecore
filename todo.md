# Production Hardening Roadmap

## Phase 1: Critical Fixes
- [ ] **[Backend] WebSocket Hub Concurrency Fix**: Resolve race condition when deleting clients from map during broadcast.
- [ ] **[Backend] Graceful Shutdown**: Implement signal handling to ensure bot stops cleanly without leaving dangling states.
- [ ] **[Backend] Persistence Layer**: Transition from in-memory state to SQLite/JSON store for active positions and balance.

## Phase 2: Observability & Configuration
- [ ] **[Backend] Structured Logging**: Replace `fmt.Printf` with `slog` or `zap` for production-grade telemetry.
- [ ] **[Backend] Configuration Validation**: Implement strict validation for trading parameters (Slippage, Fees, Risk %).
- [ ] **[Frontend] Data Integrity**: Replace hardcoded values (like $10k initial balance) with real data from the backend.

## Phase 3: UX & Reliability
- [ ] **[Frontend] Global Error Handling**: Implement Toast notifications and Error Boundaries for API/WebSocket failures.
- [ ] **[Frontend] Loading States**: Add skeleton loaders and improved feedback during "Engage Warp Sync".
- [ ] **[DevOps] Containerization**: Create a multi-stage Dockerfile for the Go backend and Vite frontend.