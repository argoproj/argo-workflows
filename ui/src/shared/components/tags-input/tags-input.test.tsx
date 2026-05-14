import {render, screen} from '@testing-library/react';
import React from 'react';

import {TagsInput} from './tags-input';

describe('TagsInput', () => {
    it('shows the full tag value on hover', () => {
        const tag = 'workflows.argoproj.io/workflow-template=workflow-template-with-a-long-name';

        render(<TagsInput tags={[tag]} onChange={jest.fn()} />);

        expect(screen.getByText(tag)).toHaveAttribute('title', tag);
    });
});
