import { expect } from 'chai';

import { Utils } from './utils';

describe('Utils', () => {
    it('returns correct short node name', () => {
        expect(Utils.shortNodeName({name: 'ci-example-kxzs4.test', displayName: 'test'})).to.be.eq('test');
        expect(Utils.shortNodeName({name: 'ci-example-kxzs4', displayName: null})).to.be.eq('ci-example-kxzs4');
    });
});
