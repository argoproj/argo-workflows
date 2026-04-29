import {escapeInvalidMarkdown} from '../../workflows/utils';

describe('escapeInvalidMarkdown', () => {
    it('escapes markdown', () => {
        expect(escapeInvalidMarkdown('**bold**')).toBe('**bold**');
        expect(escapeInvalidMarkdown('_italic_')).toBe('_italic_');
        expect(escapeInvalidMarkdown('~strikethrough~')).toBe('~strikethrough~');
        expect(escapeInvalidMarkdown('`code`')).toBe('`code`');
        expect(escapeInvalidMarkdown('[argo](https://github.com/argoproj/argo-workflows')).toBe('[argo](https://github.com/argoproj/argo-workflows');
        expect(escapeInvalidMarkdown('__underline__')).toBe('__underline__');
        expect(escapeInvalidMarkdown('```code block```')).toBe('code block');
        expect(escapeInvalidMarkdown('> quote')).toBe('quote');
        expect(escapeInvalidMarkdown('# header')).toBe('header');
        expect(escapeInvalidMarkdown('-# subheader')).toBe('subheader');
        expect(escapeInvalidMarkdown('\nthis\nis\ntext\nwith\nline\nbreaks\n')).toBe('this is text with line breaks');
        expect(escapeInvalidMarkdown('- list item\n* list item\n1. list item')).toBe('\\- list item \\* list item 1\\. list item');
    });
});
