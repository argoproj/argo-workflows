import { LoginComponent } from './views/login/login.component';
import { LayoutComponent } from './views/layout/layout.component';
import { SetupComponent } from './views/setup/setup.component';
import { SignUpComponent } from './views/setup/sign-up/sign-up.component';
import { EmailConfirmationComponent } from './views/setup/email-confirmation/email-confirmation.component';
import { ForgotPasswordComponent } from './views/forgot-password/forgot-password.component';
import { ResetPasswordComponent } from './views/reset-password/reset-password.component';
import { ResetConfirmationComponent } from './views/reset-password/reset-confirmation/reset-confirmation.component';
import { ForgotConfirmationComponent } from './views/forgot-password/forgot-confirmation/forgot-confirmation.component';

export const routes = [
    { path: 'login/:fwd', component: LoginComponent, terminal: true },
    { path: 'login', component: LoginComponent, terminal: true },
    {
        path: 'setup',
        component: SetupComponent,
        children: [
            { path: 'signup/:token', component: SignUpComponent, terminal: true },
            { path: 'confirm', component: EmailConfirmationComponent },
        ]
    },
    { path: 'forgot-password', component: ForgotPasswordComponent, terminal: true },
    { path: 'forgot-password/confirm', component: ForgotConfirmationComponent, terminal: true },
    { path: 'reset-password/confirm', component: ResetConfirmationComponent, terminal: true },
    { path: 'reset-password/:token', component: ResetPasswordComponent, terminal: true },
    { path: 'fue', loadChildren: () => System.import('./views/+first-user-experience').then((comp: any) => comp.default) },
    {
        path: 'app',
        component: LayoutComponent,
        children: [
            { path: 'search', loadChildren: () => System.import('./views/+global-search').then((comp: any) => comp.default) },
            { path: 'timeline', loadChildren: () => System.import('./views/+timeline').then((comp: any) => comp.default) },
            { path: 'integrations', loadChildren: () => System.import('./views/+integrations').then((comp: any) => comp.default) },
            { path: 'settings', loadChildren: () => System.import('./views/+settings').then((comp: any) => comp.default) },
            { path: 'cashboard', loadChildren: () => System.import('./views/+cashboard').then((comp: any) => comp.default) },
            { path: 'policies', loadChildren: () => System.import('./views/+policies').then((comp: any) => comp.default) },
            { path: 'applications', loadChildren: () => System.import('./views/+applications').then((comp: any) => comp.default) },
            { path: 'fixtures', loadChildren: () => System.import('./views/+fixtures').then((comp: any) => comp.default) },
            { path: 'service-catalog', loadChildren: () => System.import('./views/+service-catalog').then((comp: any) => comp.default) },
            { path: 'help', loadChildren: () => System.import('./views/+help').then((comp: any) => comp.default) },
            { path: 'metrics', loadChildren: () => System.import('./views/+metrics').then((comp: any) => comp.default) },
            { path: 'ax-catalog', loadChildren: () => System.import('./views/+ax-catalog').then((comp: any) => comp.default) },
            { path: 'volumes', loadChildren: () => System.import('./views/+volumes').then((comp: any) => comp.default) },
            { path: 'user-management', loadChildren: () => System.import('./views/+user-management').then((comp: any) => comp.default) },
            { path: 'infrastructure', loadChildren: () => System.import('./views/+infrastructure').then((comp: any) => comp.default) },
            { path: 'performance', loadChildren: () => System.import('./views/+performance').then((comp: any) => comp.default) },
            { path: 'hosts', loadChildren: () => System.import('./views/+hosts').then((comp: any) => comp.default) },
            { path: 'config-management', loadChildren: () => System.import('./views/+config-management').then((comp: any) => comp.default) },
            // Redirect route for backward compatibility.
            { path: 'jobs/job-details/:id', redirectTo: 'timeline/jobs/:id' },
        ]
    },
    { path: '**', redirectTo: 'login' }
];
