import {render} from '@testing-library/react';
import {Autocomplete} from 'argo-ui/src/components/autocomplete/autocomplete';
import React from 'react';

import {InputFilter} from './input-filter';

jest.mock('argo-ui/src/components/autocomplete/autocomplete', () => ({
    Autocomplete: jest.fn(() => null)
}));

const lastItems = (): string[] => {
    const mock = Autocomplete as unknown as jest.Mock;
    const lastCall = mock.mock.calls[mock.mock.calls.length - 1];
    return lastCall[0].items as string[];
};

describe('InputFilter', () => {
    beforeEach(() => {
        (Autocomplete as unknown as jest.Mock).mockClear();
        localStorage.clear();
    });

    it('passes only the localStorage cache when extraSuggestions is undefined', () => {
        localStorage.setItem('ns_inputs', 'argo,kube-system');
        render(<InputFilter value='' name='ns' onChange={() => undefined} />);
        expect(lastItems()).toEqual(['argo', 'kube-system']);
    });

    it('passes extraSuggestions ahead of the localStorage cache and dedups overlaps', () => {
        localStorage.setItem('ns_inputs', 'argo,kube-system');
        render(<InputFilter value='' name='ns' onChange={() => undefined} extraSuggestions={['prod', 'argo', 'staging']} />);
        expect(lastItems()).toEqual(['prod', 'argo', 'staging', 'kube-system']);
    });

    it('handles empty localStorage and empty extraSuggestions', () => {
        render(<InputFilter value='' name='ns' onChange={() => undefined} extraSuggestions={[]} />);
        expect(lastItems()).toEqual([]);
    });
});
