import {fireEvent, render} from '@testing-library/react';
import {Layout} from 'argo-ui/src/components/layout/layout';
import * as React from 'react';
import {MemoryRouter, useLocation} from 'react-router-dom';

// Regression test for the argo-ui nav-bar, which was patched router-agnostic during the
// react-router v7 migration (ui/patches/argo-ui+1.0.0.patch). The patched NavBar:
//   - highlights the active item by adding the `active` class to `.nav-bar__item`
//     when the current location matches that item's `path` (via useLocation), and
//   - navigates on click via useNavigate (`navigate(item.path)`).
// Layout renders NavBar, so rendering Layout exercises the real (patched) component.

// A representative subset of the nav items the workflows app passes to <Layout> in
// app-router.tsx. Each item's path is what the active-highlight and click-navigation
// behavior keys off of.
const navItems = [
    {title: 'Workflows', path: '/workflows', iconClassName: 'fa fa-stream'},
    {title: 'Workflow Templates', path: '/workflow-templates', iconClassName: 'fa fa-window-maximize'},
    {title: 'Cron Workflows', path: '/cron-workflows', iconClassName: 'fa fa-clock'}
];

// Surfaces the current pathname so the test can assert that a click navigated.
function LocationProbe() {
    const location = useLocation();
    return <span data-testid='pathname'>{location.pathname}</span>;
}

// Returns the clickable nav-bar item <div> for the given icon class. The icon class is
// the only stable, item-distinguishing marker the NavBar renders (the title is only in a
// tooltip, which Tippy does not render into the DOM until hovered).
function navItemFor(container: HTMLElement, iconClassName: string): HTMLElement {
    const icon = container.querySelector<HTMLElement>(`i.${iconClassName.split(' ').join('.')}`);
    if (!icon) {
        throw new Error(`no nav item with icon ${iconClassName}`);
    }
    return icon.closest<HTMLElement>('.nav-bar__item')!;
}

function renderNavBar(initialPath: string) {
    return render(
        <MemoryRouter initialEntries={[initialPath]}>
            <Layout navItems={navItems}>
                <LocationProbe />
            </Layout>
        </MemoryRouter>
    );
}

describe('argo-ui NavBar (react-router v7)', () => {
    it('marks the item matching the current route active and leaves the others inactive', () => {
        const {container} = renderNavBar('/workflows');

        expect(navItemFor(container, 'fa fa-stream')).toHaveClass('active');
        expect(navItemFor(container, 'fa fa-window-maximize')).not.toHaveClass('active');
        expect(navItemFor(container, 'fa fa-clock')).not.toHaveClass('active');
    });

    it('treats a sub-path of an item as active (prefix match)', () => {
        const {container} = renderNavBar('/workflows/argo/my-wf');

        // isActiveRoute matches `path` or `path/...`, so a workflow detail page keeps Workflows highlighted.
        expect(navItemFor(container, 'fa fa-stream')).toHaveClass('active');
        expect(navItemFor(container, 'fa fa-window-maximize')).not.toHaveClass('active');
    });

    it('navigates to the clicked item and moves the active highlight', () => {
        const {container, getByTestId} = renderNavBar('/workflows');
        expect(getByTestId('pathname')).toHaveTextContent('/workflows');

        fireEvent.click(navItemFor(container, 'fa fa-clock'));

        // useNavigate pushed the new route...
        expect(getByTestId('pathname')).toHaveTextContent('/cron-workflows');
        // ...and the highlight followed the location.
        expect(navItemFor(container, 'fa fa-clock')).toHaveClass('active');
        expect(navItemFor(container, 'fa fa-stream')).not.toHaveClass('active');
    });
});
