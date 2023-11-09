import {formatLabel} from './label';

const getLabels = (label: any): string[] => {
    return [label.props.children[0].props.children, label.props.children[1].props.children];
};

describe('format label', () => {
    test('wraps correctly', () => {
        const label = formatLabel('foobar-longname-1234');
        expect(getLabels(label)).toEqual(['foobar-', 'longname-1234']);
    });
    test('many possible wraps', () => {
        const label = formatLabel('many-wraps-possible-here-ok');
        expect(getLabels(label)).toEqual(['many-wraps-', 'possible-here-ok']);
    });
    test('too short to wrap', () => {
        const label = formatLabel('too-short');
        expect(label.props.children).toEqual('too-short');
    });
    test('too long to wrap', () => {
        const label = formatLabel('many-wraps-possible-here-ok-but-this-is-now-too-long-to-wrap');
        expect(getLabels(label)).toEqual([['many-wraps-p', '..'], '-long-to-wrap']);
    });
});
