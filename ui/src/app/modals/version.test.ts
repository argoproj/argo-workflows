import {majorMinor} from './version';

describe('version', () => {
    test('untagged', () => {
        expect(majorMinor('untagged')).toEqual('v0.0');
    });
    test('v0.1.0', () => {
        expect(majorMinor('v0.1.0')).toEqual('v0.1');
    });
});
