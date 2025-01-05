import {parseResourceQuantity} from './resource-quantity';

describe('resource quaniity', () => {
    test('binary SI', () => {
        expect(parseResourceQuantity('1Ki')).toEqual(1024);
        expect(parseResourceQuantity('2Ki')).toEqual(2048);
    });
    test('decimal SI', () => {
        expect(parseResourceQuantity('1m')).toEqual(1 / 1000);
        expect(parseResourceQuantity('1000m')).toEqual(1);
        expect(parseResourceQuantity('1')).toEqual(1);
        expect(parseResourceQuantity('2')).toEqual(2);
        expect(parseResourceQuantity('1k')).toEqual(1000);
        expect(parseResourceQuantity('56.633')).toEqual(56.633); // this test may not be needed as decimal point is not really valid
    });
});
