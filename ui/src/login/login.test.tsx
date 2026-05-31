import {fireEvent, render} from '@testing-library/react';
import React from 'react';
import {MemoryRouter} from 'react-router-dom';

import {deleteCookie, setCookie} from '../shared/cookie';
import {Login} from './login';

jest.mock('../shared/cookie');

describe('Login', () => {
    const LoginAt = (entry: string) => (
        <MemoryRouter initialEntries={[entry]}>
            <Login />
        </MemoryRouter>
    );

    beforeEach(() => {
        const base = document.createElement('base');
        base.setAttribute('href', '/');
        document.head.appendChild(base);
    });

    afterEach(() => {
        document.querySelector('base').remove();
    });

    describe('SSO login', () => {
        it('button has right href', () => {
            const {getAllByText} = render(LoginAt('/login'));
            const button = getAllByText('Login')[0];
            expect(button.getAttribute('href')).toBe('/oauth2/redirect?redirect=%2Fworkflows');
        });

        it('button has right href with custom <base>', () => {
            document.querySelector('base').setAttribute('href', '/test/');
            const {getAllByText} = render(LoginAt('/login'));

            const button = getAllByText('Login')[0];
            expect(button.getAttribute('href')).toBe('/test/oauth2/redirect?redirect=%2Ftest%2Fworkflows');
        });

        it('button has right href when ?redirect set', () => {
            const {getAllByText} = render(LoginAt('/login?redirect=/workflow-templates'));

            const button = getAllByText('Login')[0];
            expect(button.getAttribute('href')).toBe('/oauth2/redirect?redirect=%2Fworkflow-templates');
        });
    });

    describe('token login', () => {
        it('responds to click', () => {
            const {getAllByText, getByRole} = render(LoginAt('/login'));

            const button = getAllByText('Login')[1];
            fireEvent.change(getByRole('textbox'), {target: {value: 'test-token'}});
            fireEvent.click(button);

            expect(button.getAttribute('href')).toBe('/');
            expect(setCookie).toHaveBeenCalledWith('authorization', 'test-token');
        });

        it('responds to click with custom <base>', () => {
            document.querySelector('base').setAttribute('href', '/test/argo');
            const {getAllByText, getByRole} = render(LoginAt('/login'));

            const button = getAllByText('Login')[1];
            fireEvent.change(getByRole('textbox'), {target: {value: 'test123'}});
            fireEvent.click(button);

            expect(button.getAttribute('href')).toBe('/test/argo');
            expect(setCookie).toHaveBeenCalledWith('authorization', 'test123');
        });
    });

    describe('logout', () => {
        it('responds to button click', () => {
            // jsdom does not implement window.location.reload, so replace location with a stub
            const originalLocation = window.location;
            const reload = jest.fn();
            delete (window as any).location;
            (window as any).location = {...originalLocation, reload};

            const {getByText} = render(LoginAt('/login'));
            fireEvent.click(getByText('Logout'));
            expect(deleteCookie).toHaveBeenCalledWith('authorization');
            expect(reload).toHaveBeenCalled();

            (window as any).location = originalLocation;
        });
    });
});
