import {bearer} from '../fixtures/auth';
import {expect, test} from '../fixtures/test';

// Exercises the login page itself, so it must start unauthenticated.
test.use({storageState: {cookies: [], origins: []}});

test('logs in with a client token and lands on the workflows list', async ({page, loginPage}) => {
    await loginPage.goto();
    await loginPage.loginWithToken(bearer());
    await expect(page).toHaveURL(/\/workflows\b/);
    // A recognisable list-page control confirms we are authenticated, not bounced back to login.
    await expect(page.getByRole('button', {name: 'Submit New Workflow'})).toBeVisible();
});

test('logout clears the session', async ({page, context, loginPage}) => {
    const hasAuthCookie = async () => (await context.cookies()).some(c => c.name === 'authorization');

    await loginPage.goto();
    await loginPage.loginWithToken(bearer());
    await expect(page).toHaveURL(/\/workflows\b/);
    expect(await hasAuthCookie()).toBe(true);

    await loginPage.goto();
    const baseHRef = await page.locator('base').getAttribute('href');
    expect(baseHRef).not.toBeNull();
    const postLogoutURL = new URL(baseHRef!, page.url()).toString();
    const logoutRedirect = page.waitForResponse(response => response.url().endsWith('/auth/logout'));
    await loginPage.logout();
    const response = await logoutRedirect;
    expect(response.status()).toBe(303);
    expect(new URL(response.headers()['location'], response.url()).toString()).toBe(postLogoutURL);
    // Logout deletes the authorization cookie regardless of server auth mode.
    await expect.poll(hasAuthCookie).toBe(false);
});
