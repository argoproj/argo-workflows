import {act, fireEvent, render, screen, waitFor} from '@testing-library/react';
import {createMemoryHistory} from 'history';
import * as React from 'react';

import {Context} from '../shared/context';
import {exampleCronWorkflow} from '../shared/examples';
import {services} from '../shared/services';
import {CronWorkflowDetails} from './cron-workflow-details';

jest.mock('argo-ui/src/components/page/page', () => ({
    Page: ({children, toolbar}: any) => (
        <>
            <button onClick={toolbar.actionMenu.items.find((item: {title: string}) => item.title === 'Delete').action}>Delete</button>
            {children}
        </>
    )
}));
jest.mock('argo-ui/src/components/sliding-panel/sliding-panel', () => ({
    SlidingPanel: ({children}: any) => <>{children}</>
}));
jest.mock('../shared/use-collect-event', () => ({
    useCollectEvent: jest.fn()
}));
jest.mock('../widgets/widget-gallery', () => ({
    WidgetGallery: (): React.ReactElement => <></>
}));
jest.mock('../workflows/components/workflow-details-list/workflow-details-list', () => ({
    WorkflowDetailsList: (): React.ReactElement => <></>
}));
jest.mock('./cron-workflow-editor', () => ({
    CronWorkflowEditor: () => <div data-testid='cron-workflow-editor' />
}));

describe('CronWorkflowDetails deletion', () => {
    let testBase: HTMLBaseElement;

    beforeEach(() => {
        testBase = document.createElement('base');
        testBase.setAttribute('href', '/');
        document.head.appendChild(testBase);
    });

    afterEach(() => {
        testBase.remove();
        jest.restoreAllMocks();
    });

    async function renderDetails() {
        const cronWorkflow = exampleCronWorkflow('ns');
        cronWorkflow.metadata.name = 'cron-workflow';

        jest.spyOn(services.cronWorkflows, 'get').mockResolvedValue(cronWorkflow);
        jest.spyOn(services.cronWorkflows, 'delete').mockResolvedValue({} as any);
        jest.spyOn(services.workflows, 'list').mockResolvedValue({items: []} as any);
        jest.spyOn(services.info, 'getInfo').mockResolvedValue({columns: []} as any);

        let resolveConfirmation: (confirmed: boolean) => void;
        const confirm = jest.fn<Promise<boolean>, [string, React.ComponentType]>(
            () =>
                new Promise<boolean>(resolve => {
                    resolveConfirmation = resolve;
                })
        );
        const popup = {
            confirm
        };
        const navigation = {goto: jest.fn()};
        const history = createMemoryHistory();

        render(
            <Context.Provider value={{popup, navigation, notifications: {show: jest.fn()}, history} as any}>
                <CronWorkflowDetails history={history} location={history.location} match={{params: {name: 'cron-workflow', namespace: 'ns'}} as any} />
            </Context.Provider>
        );

        await screen.findByTestId('cron-workflow-editor');
        return {navigation, popup, resolveConfirmation: (confirmed: boolean) => resolveConfirmation(confirmed)};
    }

    it('uses cascading deletion when keep Workflows is not selected', async () => {
        const {navigation, popup, resolveConfirmation} = await renderDetails();

        fireEvent.click(screen.getByRole('button', {name: 'Delete'}));
        await act(async () => resolveConfirmation(true));

        await waitFor(() => {
            expect(services.cronWorkflows.delete).toHaveBeenCalledWith('cron-workflow', 'ns', false);
            expect(navigation.goto).toHaveBeenCalledWith('/cron-workflows/ns');
        });
        expect(popup.confirm).toHaveBeenCalledWith('Confirm', expect.any(Function));
    });

    it('uses orphan deletion when keep Workflows is selected', async () => {
        const {navigation, popup, resolveConfirmation} = await renderDetails();

        fireEvent.click(screen.getByRole('button', {name: 'Delete'}));
        const Confirmation = popup.confirm.mock.calls[0][1] as React.ComponentType;
        render(<Confirmation />);
        fireEvent.click(screen.getByLabelText('Keep Workflows created by this CronWorkflow'));
        await act(async () => resolveConfirmation(true));

        await waitFor(() => {
            expect(services.cronWorkflows.delete).toHaveBeenCalledWith('cron-workflow', 'ns', true);
            expect(navigation.goto).toHaveBeenCalledWith('/cron-workflows/ns');
        });
    });
});
