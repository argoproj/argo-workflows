import {Locator, Page} from '@playwright/test';

// Page object for ui/src/login/login.tsx. The token section stores whatever is
// pasted as the `authorization` cookie; the SSO section is out of scope.
export class LoginPage {
    readonly tokenInput: Locator;
    readonly loginButton: Locator;
    readonly logoutButton: Locator;

    constructor(private readonly page: Page) {
        this.tokenInput = page.locator('#token');
        this.loginButton = page.locator('.login__token-section').getByRole('link', {name: 'Login'});
        // The logout anchor has no href, so it has no `link` role — match by text.
        this.logoutButton = page.locator('.login__logout-section').getByText('Logout');
    }

    async goto(): Promise<void> {
        await this.page.goto('login');
    }

    /** `token` must include the `Bearer ` prefix, exactly as a user would paste it. */
    async loginWithToken(token: string): Promise<void> {
        await this.tokenInput.fill(token);
        await this.loginButton.click();
    }

    async logout(): Promise<void> {
        await this.logoutButton.click();
    }
}
