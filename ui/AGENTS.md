# UI

React 18 + TypeScript, webpack, rxjs observables (no Redux), `argo-ui` component library pinned to a git SHA. Feature areas map 1:1 to `src/` directories; API clients live in `src/shared/services`. `src/shared/pod-name.ts` must stay in sync with the controller's pod naming.

- Dev server: `yarn start` (:8080) proxies API and artifact routes to argo-server on :2746 (usually from `make start`).
- Tests: `yarn test` (jest). E2E: `make test-ui-e2e` (playwright; needs `make start AUTH_MODE=client`; first run: `yarn playwright install --with-deps chromium`).
- Lint: `yarn lint` (eslint `--fix` on `src` and `e2e`, then `tsc --noEmit` for both the app and `e2e/tsconfig.json`) — commit what `--fix` changes.
- The production bundle is embedded into the server binary via `ui/embed.go`.
