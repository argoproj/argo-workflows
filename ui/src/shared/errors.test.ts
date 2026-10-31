import {getErrorMessage} from './errors';

describe('getErrorMessage', () => {
    it('returns the message of a plain error', () => {
        expect(getErrorMessage(new Error('Workflow gone'))).toBe('Workflow gone');
    });

    it('includes the message from a HTTP error body', () => {
        const error = Object.assign(new Error('Forbidden'), {response: {body: {code: 7, message: 'Permission denied'}}});
        expect(getErrorMessage(error)).toBe('Forbidden: Permission denied');
    });

    it('does not show the raw body of a HTTP error without a status text', () => {
        // super-agent's message for a HTTP/2 response is the response text
        const text = '{"code":7,"message":"Permission denied"}';
        const error = Object.assign(new Error(text), {response: {text, body: JSON.parse(text)}});
        expect(getErrorMessage(error)).toBe('Permission denied');
    });

    it('falls back when there is no message', () => {
        expect(getErrorMessage(new Error())).toMatch(/^Unknown error/);
        expect(getErrorMessage(undefined)).toMatch(/^Unknown error/);
    });
});
