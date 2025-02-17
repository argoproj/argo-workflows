import {majorMinor} from './version';

describe('version', () => {
    test('untagged', () => {
        expect(majorMinor('untagged')).toEqual('untagged');
    });
    test('v0.1.0', () => {
        expect(majorMinor('v0.1.0')).toEqual('v0.1');
    });
});
