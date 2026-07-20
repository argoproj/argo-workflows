import * as path from 'path';

// Shared filesystem locations, resolved relative to this file so they are
// independent of the working directory Playwright happens to run from.
export const AUTH_DIR = path.join(__dirname, '.auth');
export const STORAGE_STATE = path.join(AUTH_DIR, 'state.json');
