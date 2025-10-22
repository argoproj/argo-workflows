import {fireEvent, render} from '@testing-library/react';
import {createMemoryHistory, History} from 'history';
import React from 'react';

import {deleteCookie, setCookie} from '../shared/cookie';
import {Login} from './login';

jest.mock('../shared/cookie');

describe('Login', () => {
    const LoginWithHistory = (history: History) => <Login history={history} match={null} location={history.location} />;

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
            const {getAllByText} = render(LoginWithHistory(createMemoryHistory()));
            const button = getAllByText('Login')[3];
            expect(button.getAttribute('href')).toBe('/oauth2/redirect?redirect=%2Fworkflows');
        });

        it('button has right href with custom <base>', () => {
            document.querySelector('base').setAttribute('href', '/test/');
            const {getAllByText} = render(LoginWithHistory(createMemoryHistory()));

            const button = getAllByText('Login')[3];
            expect(button.getAttribute('href')).toBe('/test/oauth2/redirect?redirect=%2Ftest%2Fworkflows');
        });

        it('button has right href when ?redirect set', () => {
            const history = createMemoryHistory();
            history.push('/login?redirect=/workflow-templates');
            const {getAllByText} = render(LoginWithHistory(history));

            const button = getAllByText('Login')[3];
            expect(button.getAttribute('href')).toBe('/oauth2/redirect?redirect=%2Fworkflow-templates');
        });
    });

    describe('token login', () => {
        it('responds to click', () => {
            const {getAllByText, getByRole} = render(LoginWithHistory(createMemoryHistory()));

            const button = getAllByText('Login')[4];
            fireEvent.change(getByRole('textbox'), {target: {value: 'test-token'}});
            fireEvent.click(button);

            expect(button.getAttribute('href')).toBe('/');
            expect(setCookie).toHaveBeenCalledWith('authorization', 'test-token');
        });

        it('responds to click with custom <base>', () => {
            document.querySelector('base').setAttribute('href', '/test/argo');
            const {getAllByText, getByRole} = render(LoginWithHistory(createMemoryHistory()));

            const button = getAllByText('Login')[4];
            fireEvent.change(getByRole('textbox'), {target: {value: 'test123'}});
            fireEvent.click(button);

            expect(button.getAttribute('href')).toBe('/test/argo');
            expect(setCookie).toHaveBeenCalledWith('authorization', 'test123');
        });
    });

    describe('logout', () => {
        it('responds to button click', () => {
            const {getByText} = render(LoginWithHistory(createMemoryHistory()));
            fireEvent.click(getByText('Logout'));
            expect(deleteCookie).toHaveBeenCalledWith('authorization');
        });
    });
});
