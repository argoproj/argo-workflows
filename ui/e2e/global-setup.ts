import {mkdirSync, writeFileSync} from 'fs';

import {majorMinor} from '../src/modals/version';
import {BASE_URL, bearer} from './fixtures/auth';
import {AUTH_DIR, STORAGE_STATE} from './paths';

// Seeding modal/version needs the *actual* server version so it matches what the
// app computes; an empty/guessed value would let the new-version modal overlay
// tests. The version endpoint sits behind the gatekeeper (no per-method
// exemption), so it needs the bearer token under --auth-mode=client (as CI runs).
// Retry briefly to tolerate a stack that is still coming up, then fail loudly —
// the whole suite needs the server, so a clear setup error beats a silent flake.
async function serverVersion(): Promise<string> {
    let lastErr: unknown;
    for (let attempt = 0; attempt < 5; attempt++) {
        try {
            const res = await fetch(`${BASE_URL}/api/v1/version`, {headers: {Authorization: bearer()}});
            if (res.ok) {
                const version = (await res.json()).version;
                if (version) {
                    return majorMinor(version);
                }
                lastErr = new Error('version field was empty');
            } else {
                lastErr = new Error(`HTTP ${res.status}`);
            }
        } catch (err) {
            lastErr = err;
        }
        await new Promise(resolve => setTimeout(resolve, 2_000));
    }
    throw new Error(`could not fetch ${BASE_URL}/api/v1/version to seed modal suppression (is the stack up?): ${lastErr}`);
}

// Runs once before the suite. Rather than driving the login page for every test,
// we mint a Playwright storage state that (a) carries the `authorization` cookie
// Login (ui/src/login/login.tsx) would set, and (b) pre-dismisses the app's
// first-time-user and new-version modals (ui/src/modals/modal-switch.tsx) so they
// never overlay a test. The feedback modal needs no seed: the app defaults it two
// weeks out, and its localStorage key can't be seeded anyway — ScopedLocalStorage
// discards a stored value whose type differs from the `null` default. login.spec.ts
// opts out of this state to cover login itself.
export default async function globalSetup(): Promise<void> {
    const url = new URL(BASE_URL);
    const version = await serverVersion();
    const state = {
        cookies: [
            {
                name: 'authorization',
                value: bearer(),
                domain: url.hostname,
                path: '/',
                expires: -1, // session cookie
                httpOnly: false,
                secure: false,
                sameSite: 'Strict' as const
            }
        ],
        origins: [
            {
                origin: url.origin,
                localStorage: [
                    {name: 'modal/ftu', value: '"dismissed"'},
                    {name: 'modal/version', value: JSON.stringify(version)}
                ]
            }
        ]
    };
    mkdirSync(AUTH_DIR, {recursive: true});
    // Consumed by Playwright via `storageState` in playwright.config.ts, not imported anywhere.
    writeFileSync(STORAGE_STATE, JSON.stringify(state, null, 2));
}
