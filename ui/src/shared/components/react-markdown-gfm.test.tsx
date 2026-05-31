import {render, screen} from '@testing-library/react';
import React from 'react';

import {ReactMarkdownGfm} from './_react-markdown-gfm';

describe('ReactMarkdownGfm', () => {
    it('renders a GFM table and link without throwing', () => {
        const markdown = ['[argo](https://github.com/argoproj/argo-workflows)', '', '| a | b |', '| - | - |', '| 1 | 2 |'].join('\n');

        render(<ReactMarkdownGfm markdown={markdown} />);

        // remark-gfm turns the pipe syntax into a real <table>
        expect(screen.getByRole('table')).toBeInTheDocument();
        expect(screen.getByRole('cell', {name: '1'})).toBeInTheDocument();

        // the custom anchor component renders the link text + href
        const link = screen.getByRole('link', {name: 'argo'});
        expect(link).toHaveAttribute('href', 'https://github.com/argoproj/argo-workflows');
    });
});
