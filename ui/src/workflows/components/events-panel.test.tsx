import {fireEvent, render, screen, waitFor} from '@testing-library/react';
import React from 'react';
import {NEVER, Observable} from 'rxjs';

import {services} from '../../shared/services';
import {EventsPanel} from './events-panel';

afterEach(() => jest.restoreAllMocks());

it('switches subscriptions when a same-name Workflow has a different UID', async () => {
    const closed = jest.fn();
    const watch = jest.spyOn(services.workflows, 'watchEvents').mockImplementation((_, selector) => new Observable(() => () => closed(selector)));
    const props = {namespace: 'argo', kind: 'Workflow', name: 'reused'};
    const selector = 'involvedObject.kind=Workflow,involvedObject.name=reused,involvedObject.uid=';
    const {rerender} = render(<EventsPanel {...props} uid='old' />);
    await waitFor(() => expect(watch).toHaveBeenLastCalledWith('argo', selector + 'old'));

    rerender(<EventsPanel {...props} uid='new' />);
    await waitFor(() => expect(watch).toHaveBeenLastCalledWith('argo', selector + 'new'));
    expect(closed).toHaveBeenCalledWith(selector + 'old');

    fireEvent.click(screen.getByText('Show All'));
    await waitFor(() => expect(watch).toHaveBeenLastCalledWith('argo', ''));
    expect(closed).toHaveBeenCalledWith(selector + 'new');
});

it('watches a Pod by name when its UID is unavailable', async () => {
    const watch = jest.spyOn(services.workflows, 'watchEvents').mockReturnValue(NEVER);
    render(<EventsPanel namespace='argo' kind='Pod' name='worker' />);
    await waitFor(() => expect(watch).toHaveBeenCalledWith('argo', 'involvedObject.kind=Pod,involvedObject.name=worker'));
});
