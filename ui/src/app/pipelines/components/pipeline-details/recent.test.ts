import {recent} from './recent';

describe('recent', () => {
    test('recency', () => {
        expect(recent(null)).toEqual(false);
        expect(recent(new Date())).toEqual(true);
        expect(recent(new Date(0))).toEqual(false);
    });
});
