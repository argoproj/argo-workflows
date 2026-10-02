import {render, screen} from '@testing-library/react';
import React from 'react';

import {LinkButtons} from './links';

describe('LinkButtons', () => {
    it('renders links as anchors, so they can be opened in a new tab from the context menu', () => {
        render(<LinkButtons links={[{name: 'Logs', scope: 'workflow-list', url: 'https://logging/${metadata.name}', target: '_self'}]} />);
        const link = screen.getByRole('link', {name: 'Logs'});
        expect(link.tagName).toBe('A');
        // without an object, the URL is not templated
        expect(link).toHaveAttribute('href', 'https://logging/${metadata.name}');
        expect(link).toHaveAttribute('target', '_self');
    });

    it('templates the URL with the object and defaults to opening in a new tab', () => {
        const object = {metadata: {namespace: 'argo', name: 'my-wf'}, status: {}};
        render(<LinkButtons links={[{name: 'Logs', scope: 'workflow', url: 'https://logging/${metadata.namespace}/${metadata.name}', target: ''}]} object={object} />);
        const link = screen.getByRole('link', {name: 'Logs'});
        expect(link).toHaveAttribute('href', 'https://logging/argo/my-wf');
        expect(link).toHaveAttribute('target', '_blank');
    });
});
