import * as React from 'react';
import {ZeroState} from '../../../shared/components/zero-state';

export const EventsZeroState = (props: {title: string}) => (
    <ZeroState title={props.title}>
        <p>Argo Events allow you to trigger workflows, lambadas, and other actions based on receiving events from things like webhooks, message, or a cron schedule.</p>
        <p>
            The "mark activations" buttons allows you to see when an entity "activates". This is determined by it writing a log entry. Helpful to debug when things are happening.
        </p>
        <p>
            <a href='https://argoproj.github.io/argo-events/'>Learn more</a>
        </p>
    </ZeroState>
);
