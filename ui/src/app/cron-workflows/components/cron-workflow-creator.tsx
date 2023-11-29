import * as React from 'react';
import {useState} from 'react';
import {CronWorkflow} from '../../../models';
import {Button} from '../../shared/components/button';
import {ErrorNotice} from '../../shared/components/error-notice';
import {ExampleManifests} from '../../shared/components/example-manifests';
import {UploadButton} from '../../shared/components/upload-button';
import {exampleCronWorkflow} from '../../shared/examples';
import {services} from '../../shared/services';
import {Utils} from '../../shared/utils';
import {CronWorkflowEditor} from './cron-workflow-editor';

export function CronWorkflowCreator({onCreate, namespace}: {namespace: string; onCreate: (cronWorkflow: CronWorkflow) => void}) {
    const [cronWorkflow, setCronWorkflow] = useState<CronWorkflow>(exampleCronWorkflow(Utils.getNamespaceWithDefault(namespace)));
    const [error, setError] = useState<Error>();
    return (
        <>
            <div>
                <UploadButton onUpload={setCronWorkflow} onError={setError} />
                <Button
                    icon='plus'
                    onClick={() => {
                        services.cronWorkflows
                            .create(cronWorkflow, Utils.getNamespaceWithDefault(cronWorkflow.metadata.namespace))
                            .then(onCreate)
                            .catch(setError);
                    }}>
                    Create
                </Button>
            </div>
            <ErrorNotice error={error} />
            <CronWorkflowEditor cronWorkflow={cronWorkflow} onChange={setCronWorkflow} onError={setError} />
            <p>
                <ExampleManifests />.
            </p>
        </>
    );
}
