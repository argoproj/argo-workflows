import * as React from 'react';
import {useState} from 'react';
import {Sensor} from '../../../models';
import {Button} from '../../shared/components/button';
import {ErrorNotice} from '../../shared/components/error-notice';
import {UploadButton} from '../../shared/components/upload-button';
import {exampleSensor} from '../../shared/examples';
import {services} from '../../shared/services';
import {Utils} from '../../shared/utils';
import {SensorEditor} from './sensor-editor';

export const SensorCreator = ({namespace, onCreate}: {namespace: string; onCreate: (sensor: Sensor) => void}) => {
    const [sensor, setSensor] = useState<Sensor>(exampleSensor(Utils.getNamespaceWithDefault(namespace)));
    const [error, setError] = useState<Error>();
    return (
        <>
            <div>
                <UploadButton onUpload={setSensor} onError={setError} />
                <Button
                    icon='plus'
                    onClick={() => {
                        services.sensor
                            .create(sensor, Utils.getNamespaceWithDefault(sensor.metadata.namespace))
                            .then(onCreate)
                            .catch(setError);
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
};
