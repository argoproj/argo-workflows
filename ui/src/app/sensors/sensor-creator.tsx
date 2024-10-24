import * as React from 'react';
import {useState} from 'react';

import {Sensor} from '../../models';
import {Button} from '../shared/components/button';
import {ErrorNotice} from '../shared/components/error-notice';
import {UploadButton} from '../shared/components/upload-button';
import {exampleSensor} from '../shared/examples';
import * as nsUtils from '../shared/namespaces';
import {services} from '../shared/services';
import {SensorEditor} from './sensor-editor';

export function SensorCreator({namespace, onCreate}: {namespace: string; onCreate: (sensor: Sensor) => void}) {
    const [sensor, setSensor] = useState<Sensor>(exampleSensor(nsUtils.getNamespaceWithDefault(namespace)));
    const [error, setError] = useState<Error>();
    return (
        <>
            <div>
                <UploadButton onUpload={setSensor} onError={setError} />
                <Button
                    icon='plus'
                    onClick={async () => {
                        try {
                            const newSensor = await services.sensor.create(sensor, nsUtils.getNamespaceWithDefault(sensor.metadata.namespace));
                            onCreate(newSensor);
                        } catch (err) {
                            setError(err);
                        }
                    }}>
                    Create
                </Button>
            </div>
            <ErrorNotice error={error} />
            <SensorEditor sensor={sensor} onChange={setSensor} onError={setError} />
            <p>
                <a href='https://github.com/argoproj/argo-events/tree/stable/examples/sensors'>
                    Example sensors <i className='fa fa-external-link-alt' />
                </a>
            </p>
        </>
    );
}
