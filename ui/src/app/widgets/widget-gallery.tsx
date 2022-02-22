import * as React from 'react';
import {uiUrl} from '../shared/base';

export const WidgetGallery = ({namespace, name, label}: {namespace: string; name?: string; label?: string}) => (
    <div className='white-box'>
        <h3>Widgets</h3>
        <p>
            These widgets can be embedded inside other applications. <a href='https://argoproj.github.io/argo-workflows/widgets'>Learn more</a>
        </p>
        {[
            {
                title: 'Status Badge',
                description: 'An embeddable status badge that automatically updates when the workflow completes.',
                path: `widgets/workflow-status-badges/${namespace}?${name && 'name=' + name}&${label && 'label=' + label}`,
                parameters: [
                    {
                        name: 'target',
                        defaultValue: '_top',
                        description: 'Which frame to open the workflow in when clicked.'
                    }
                ],
                width: 200,
                height: 20
            },
            {
                title: 'Graph',
                description: `A graph of the workflow that automatically updates when nodes change state.`,
                path: `widgets/workflow-graphs/${namespace}?${name && 'name=' + name}&${label && 'label=' + label}`,
                parameters: [
                    {
                        name: 'target',
                        defaultValue: '_top',
                        description: 'Which frame to open the workflow in when clicked.'
                    },
                    {name: 'showOptions', defaultValue: 'false', description: 'Show the rendering options'},
                    {name: 'nodeSize', defaultValue: '32', description: 'The size of the nodes'}
                ],
                width: 400,
                height: 200
            }
        ].map(({title, description, width, height, path, parameters}) => (
            <>
                <h5>{title}</h5>
                <p>{description}</p>
                <div style={{margin: '20px'}}>
                    <iframe frameBorder={0} width={width} height={height} src={uiUrl(path)} />
                </div>
                <ul>
                    {parameters.map(p => (
                        <li key={p.name}>
                            {p.name}: {p.description} {!!p.defaultValue && '(default "' + p.defaultValue + '")'}
                        </li>
                    ))}
                </ul>
                <p>
                    <a href={uiUrl(path)} target='widget_preview'>
                        Preview <i className='fas fa-external-link-alt' />
                    </a>
                </p>
            </>
        ))}
        <p>
            Widgets not loading? Check for <a href='https://argoproj.github.io/argo-workflows/widgets'>solutions in the docs</a>.
        </p>
    </div>
);
