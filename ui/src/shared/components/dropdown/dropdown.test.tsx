import {fireEvent, render, screen} from '@testing-library/react';
import React from 'react';

import {DropDown} from './dropdown';

describe('DropDown', () => {
    beforeEach(() => {
        Object.defineProperty(HTMLElement.prototype, 'offsetHeight', {configurable: true, value: 50});
        Object.defineProperty(HTMLElement.prototype, 'offsetWidth', {configurable: true, value: 100});
        Object.defineProperty(HTMLElement.prototype, 'getBoundingClientRect', {
            configurable: true,
            value: () => ({top: 0, left: 0, bottom: 50, right: 100, width: 100, height: 50})
        });
    });

    // DropDown reads ref.current eagerly in the component body (not in useEffect).
    // On the initial render, refs are null so open() is a no-op.
    // Calling rerender() with a *new* JSX element forces React to re-render
    // the component, at which point the refs are populated.
    function renderDropDown(anchor: React.ReactElement | ((opened: boolean) => React.ReactElement), children?: React.ReactNode) {
        const result = render(<DropDown anchor={anchor}>{children || <div>Content</div>}</DropDown>);
        result.rerender(<DropDown anchor={anchor}>{children || <div>Content</div>}</DropDown>);
        return result;
    }

    it('renders anchor as JSX element', () => {
        renderDropDown(<button>Open</button>);
        expect(screen.getByText('Open')).toBeInTheDocument();
    });

    it('renders anchor as function and passes opened=false initially', () => {
        const anchorFn = jest.fn((opened: boolean) => <button>{opened ? 'Close' : 'Open'}</button>);
        renderDropDown(anchorFn);
        expect(anchorFn).toHaveBeenCalledWith(false);
        expect(screen.getByText('Open')).toBeInTheDocument();
    });

    it('opens dropdown on anchor click and passes opened=true to anchor function', () => {
        const anchorFn = jest.fn((opened: boolean) => <button>{opened ? 'Close' : 'Open'}</button>);
        renderDropDown(anchorFn);

        fireEvent.click(screen.getByText('Open'));

        expect(anchorFn).toHaveBeenLastCalledWith(true);
        expect(screen.getByText('Close')).toBeInTheDocument();
    });

    it('closes dropdown when anchor is clicked while open (toggle behavior)', () => {
        const anchorFn = jest.fn((opened: boolean) => <button data-testid='anchor'>{opened ? 'Close' : 'Open'}</button>);
        renderDropDown(anchorFn);

        fireEvent.click(screen.getByTestId('anchor'));
        expect(screen.getByText('Close')).toBeInTheDocument();

        fireEvent.click(screen.getByTestId('anchor'));
        expect(screen.getByText('Open')).toBeInTheDocument();
        expect(anchorFn).toHaveBeenLastCalledWith(false);
    });

    it('opens dropdown with JSX element anchor on click', () => {
        renderDropDown(<button>Toggle</button>);

        const content = document.querySelector('.argo-dropdown__content');
        expect(content).not.toHaveClass('opened');

        fireEvent.click(screen.getByText('Toggle'));
        expect(content).toHaveClass('opened');
    });

    it('toggles closed with JSX element anchor on second click', () => {
        renderDropDown(<button>Toggle</button>);

        const content = document.querySelector('.argo-dropdown__content');
        fireEvent.click(screen.getByText('Toggle'));
        expect(content).toHaveClass('opened');

        fireEvent.click(screen.getByText('Toggle'));
        expect(content).not.toHaveClass('opened');
    });
});
