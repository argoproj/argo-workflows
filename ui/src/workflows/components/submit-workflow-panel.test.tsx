import {act, fireEvent, render, screen} from '@testing-library/react';
import {createMemoryHistory} from 'history';
import React from 'react';

import {flush, installMockXHR, makeFile, MockXHR} from '../../shared/components/artifacts-input/artifacts-input-test-utils';
import {Context, ContextApis} from '../../shared/context';
import {services} from '../../shared/services';
import {SubmitWorkflowPanel} from './submit-workflow-panel';

type PanelProps = React.ComponentProps<typeof SubmitWorkflowPanel>;

describe('SubmitWorkflowPanel input artifacts', () => {
    const context = {navigation: {goto: jest.fn()}} as unknown as ContextApis;
    const baseProps: PanelProps = {
        kind: 'ClusterWorkflowTemplate',
        namespace: 'argo',
        name: 'my-cwft',
        entrypoint: 'main',
        templates: [],
        workflowParameters: [],
        workflowArtifacts: [{name: 'input-artifact'}],
        history: createMemoryHistory()
    };
    const panel = (props: Partial<PanelProps> = {}) => (
        <Context.Provider value={context}>
            <SubmitWorkflowPanel {...baseProps} {...props} />
        </Context.Provider>
    );
    const noticeText = 'Namespace changed. Upload the input artifacts again.';
    const uploadedText = /Uploaded: uploads\/argo\/abc\/input\.zip/;

    let submit: jest.SpyInstance;

    beforeEach(() => {
        submit = jest.spyOn(services.workflows, 'submit').mockResolvedValue({metadata: {namespace: 'other', name: 'wf'}} as never);
    });

    afterEach(() => {
        jest.restoreAllMocks();
    });

    const selectFile = (container: HTMLElement) => {
        const fileInput = container.querySelector('input[type="file"]') as HTMLInputElement;
        fireEvent.change(fileInput, {target: {files: [makeFile()]}});
    };

    const completeUpload = async (xhr: MockXHR) => {
        xhr.status = 200;
        xhr.responseText = JSON.stringify({name: 'input-artifact', key: 'uploads/argo/abc/input.zip'});
        act(() => {
            xhr.onload?.();
        });
        await flush();
    };

    it('WorkflowTemplate: upload URL has no clusterScope query', () => {
        const xhr = installMockXHR();
        const {container} = render(panel({kind: 'WorkflowTemplate', name: 'my-template'}));

        selectFile(container);

        expect(xhr.open).toHaveBeenCalledWith('POST', '/upload-artifacts/argo/my-template/input-artifact');
    });

    it('ClusterWorkflowTemplate: upload URL has clusterScope=true', () => {
        const xhr = installMockXHR();
        const {container} = render(panel({kind: 'ClusterWorkflowTemplate'}));

        selectFile(container);

        expect(xhr.open).toHaveBeenCalledWith('POST', '/upload-artifacts/argo/my-cwft/input-artifact?clusterScope=true');
    });

    it('shows no Input Artifacts section when the template has no artifacts', () => {
        installMockXHR();
        render(panel({workflowArtifacts: []}));

        expect(screen.queryByText('Input Artifacts')).toBeNull();
    });

    it.each([[undefined], ['']])('offers the upload only once the namespace is known (namespace %p)', namespace => {
        installMockXHR();
        const {container, rerender} = render(panel({namespace}));

        expect(container.querySelector('input[type="file"]')).toBeNull();
        expect(screen.queryByText('Input Artifacts')).toBeNull();

        rerender(panel({namespace: 'argo'}));

        expect(container.querySelector('input[type="file"]')).not.toBeNull();
    });

    it('namespace change discards uploaded artifacts and asks to upload again', async () => {
        const xhr = installMockXHR();
        const {container, rerender} = render(panel());

        selectFile(container);
        await completeUpload(xhr);
        expect(screen.getByText(uploadedText)).toBeTruthy();
        expect(screen.queryByText(noticeText)).toBeNull();

        rerender(panel({namespace: 'other'}));

        expect(screen.queryByText(uploadedText)).toBeNull();
        expect(screen.getByText(noticeText)).toBeTruthy();

        fireEvent.click(screen.getByText('Submit'));
        await flush();
        expect(submit).toHaveBeenCalledWith('ClusterWorkflowTemplate', 'my-cwft', 'other', expect.objectContaining({artifacts: undefined}));

        selectFile(container);
        await completeUpload(xhr);
        expect(screen.getByText(uploadedText)).toBeTruthy();
        expect(screen.queryByText(noticeText)).toBeNull();
    });

    it('namespace change with nothing uploaded shows no notice', () => {
        installMockXHR();
        const {rerender} = render(panel());

        rerender(panel({namespace: 'other'}));

        expect(screen.queryByText(noticeText)).toBeNull();
    });

    it('namespace change during upload aborts it and keeps Submit usable', () => {
        const xhr = installMockXHR();
        const {container, rerender} = render(panel());

        selectFile(container);
        const uploading = screen.getByText('Uploading...').closest('button') as HTMLButtonElement;
        expect(uploading.disabled).toBe(true);

        rerender(panel({namespace: 'other'}));

        expect(xhr.abort).toHaveBeenCalledTimes(1);
        const submitButton = screen.getByText('Submit').closest('button') as HTMLButtonElement;
        expect(submitButton.disabled).toBe(false);
    });
});
