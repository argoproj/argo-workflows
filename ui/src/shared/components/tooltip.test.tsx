import {fireEvent, render, screen} from '@testing-library/react';
import React from 'react';

import * as links from './links';
import {Tooltip} from './tooltip';

// The global jest.config.js aliases react-markdown to a fragment stub, which
// hides regressions in how the wrapper wires plugins and the <a> override.
// Overriding it locally lets us assert the contract the wrapper depends on.
const ReactMarkdownSpy = jest.fn();
jest.mock('react-markdown', () => ({
    __esModule: true,
    default: (props: {children: React.ReactNode; components?: {a?: React.ComponentType<React.ComponentProps<'a'>>}}) => {
        ReactMarkdownSpy(props);
        const Anchor = props.components?.a;
        return (
            <div data-testid='markdown'>
                <span>{props.children}</span>
                {Anchor ? (
                    <>
                        <Anchor href='https://example.com/x'>with-href</Anchor>
                        <Anchor>no-href</Anchor>
                    </>
                ) : null}
            </div>
        );
    }
}));

const argoTooltipSpy = jest.fn();
jest.mock('argo-ui/src/components/tooltip/tooltip', () => ({
    Tooltip: (props: {content: React.ReactNode; maxWidth?: string | number; children: React.ReactNode}) => {
        argoTooltipSpy(props);
        return (
            <div data-testid='argo-tooltip' data-max-width={props.maxWidth === undefined ? 'unset' : String(props.maxWidth)}>
                <div data-testid='content'>{props.content}</div>
                {props.children}
            </div>
        );
    }
}));

describe('Tooltip', () => {
    beforeEach(() => {
        ReactMarkdownSpy.mockClear();
        argoTooltipSpy.mockClear();
    });

    it('renders string content through ReactMarkdown with GFM and line-break plugins', () => {
        render(
            <Tooltip content='**hi**'>
                <span>trigger</span>
            </Tooltip>
        );

        expect(ReactMarkdownSpy).toHaveBeenCalledTimes(1);
        const mdProps = ReactMarkdownSpy.mock.calls[0][0];
        expect(mdProps.children).toBe('**hi**');
        expect(Array.isArray(mdProps.remarkPlugins)).toBe(true);
        expect(mdProps.remarkPlugins).toHaveLength(2);
        expect(mdProps.components.a).toBeDefined();
    });

    it('caps markdown tooltips at maxWidth 50vw without forcing a minimum width', () => {
        render(
            <Tooltip content='some string'>
                <span>trigger</span>
            </Tooltip>
        );
        expect(screen.getByTestId('argo-tooltip').getAttribute('data-max-width')).toBe('50vw');
        const tippyProps = argoTooltipSpy.mock.calls[0][0];
        expect(tippyProps.onCreate).toBeUndefined();
    });

    it('leaves non-string content untouched and does not force maxWidth', () => {
        render(
            <Tooltip
                content={
                    <span data-testid='rich'>
                        rich <b>node</b>
                    </span>
                }>
                <span>trigger</span>
            </Tooltip>
        );
        expect(ReactMarkdownSpy).not.toHaveBeenCalled();
        expect(screen.getByTestId('rich')).toBeInTheDocument();
        expect(screen.getByTestId('argo-tooltip').getAttribute('data-max-width')).toBe('unset');
    });

    it('opens defined hrefs via openLinkWithKey and ignores undefined hrefs', () => {
        const spy = jest.spyOn(links, 'openLinkWithKey').mockImplementation(() => undefined);
        render(
            <Tooltip content='irrelevant'>
                <span>trigger</span>
            </Tooltip>
        );

        fireEvent.click(screen.getByText('with-href'));
        expect(spy).toHaveBeenCalledWith('https://example.com/x');

        spy.mockClear();
        fireEvent.click(screen.getByText('no-href'));
        expect(spy).not.toHaveBeenCalled();

        spy.mockRestore();
    });
});
