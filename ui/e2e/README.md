# UI E2E tests (Playwright)

Browser-level end-to-end tests for the Argo Workflows UI. Unlike the Jest unit
tests (`yarn test`), these drive a real browser against a live stack.

## Running locally

Start the dev stack in one terminal (Tilt serves the UI on http://localhost:8080):

```bash
make start AUTH_MODE=client 
```

Then, in another terminal:

```bash
yarn --cwd ui install
yarn --cwd ui playwright install --with-deps chromium   # first run only
yarn --cwd ui e2e                            # headless
yarn --cwd ui e2e:ui                         # interactive/headed debugging
yarn --cwd ui playwright show-report         # open the last HTML report
```

## NixOS

Playwright's bundled browsers are dynamically linked against a standard FHS and
fail to start on NixOS (`error while loading shared libraries: libglib-2.0.so.0`).
NixOS has `nix-ld`, so provide the browser's libraries via `NIX_LD_LIBRARY_PATH`
and the bundled binary runs as-is (this covers headless, `--headed`, and `--ui`
mode, whose UI shell is also a bundled browser):

```bash
export NIX_LD_LIBRARY_PATH="$(nix eval --impure --raw --expr 'with import <nixpkgs> {}; lib.makeLibraryPath [
  glib nspr nss atk at-spi2-atk at-spi2-core cups dbus expat libdrm mesa libgbm
  libxkbcommon pango cairo alsa-lib fontconfig freetype gtk3 gdk-pixbuf systemdLibs
  xorg.libX11 xorg.libXcomposite xorg.libXdamage xorg.libXext xorg.libXfixes
  xorg.libXrandr xorg.libXrender xorg.libXtst xorg.libXi xorg.libxcb xorg.libxshmfence
]')${NIX_LD_LIBRARY_PATH:+:$NIX_LD_LIBRARY_PATH}"
yarn --cwd ui e2e:ui
```

Add that export to your dev shell (`shell.nix` / direnv) so it is always set.

If you run the dev stack inside a **devcontainer** (VS Code + Tilt), the browser
running on your host reaches the UI through a forwarded port. The argo-server
rejects workflow API calls that cross the container boundary (you'll see the UI
load but `401` on `/api/v1/workflows`). Run the browser where the API is reached
cleanly instead: either run the suite inside the devcontainer, or run a host-side
`yarn --cwd ui start` against a host `kubectl port-forward svc/argo-server 2746:2746`
and point the tests at it with `ARGO_UI_BASE_URL`.

## How it works

- **Auth** (`e2e/global-setup.ts`): reads the `argo-server` service-account token
  (the same secret the Go e2e suite uses) and writes a Playwright storage state
  containing the `authorization` cookie, so tests start logged in. Override the
  token with `ARGO_TOKEN`, or point at a different UI with `ARGO_UI_BASE_URL`.
- **Backend state** (`e2e/fixtures/api.ts`): tests seed workflows over the REST
  API and wait for a terminal phase *before* asserting on the rendered page, so
  rendering never races the controller. Created workflows are cleaned up on
  teardown.
- **Page objects** (`e2e/pages/`) centralise selectors. Prefer role/text/href
  locators; add a `data-testid` only when nothing stable exists.

Timeouts scale by `E2E_ENV_FACTOR` (set to `2` in CI) to absorb runner
contention.
