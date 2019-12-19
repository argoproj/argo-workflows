import { AppContext, DataLoader, MockupList, Page, TopBarFilter } from 'argo-ui';
import * as PropTypes from 'prop-types';
import * as React from 'react';
import { Link, RouteComponentProps } from 'react-router-dom';
import { Observable } from 'rxjs';

import * as models from '../../../../models';
import { uiUrl } from '../../../shared/base';
import { services } from '../../../shared/services';

import { WorkflowListItem } from '../workflow-list-item/workflow-list-item';

export class WorkflowsList extends React.Component<RouteComponentProps<any>> {

    public static contextTypes = {
        router: PropTypes.object,
        apis: PropTypes.object,
    };

    private get phases() {
        return new URLSearchParams(this.props.location.search).getAll('phase');
    }

    public render() {
        const filter: TopBarFilter<string> = {
            items: Object.keys(models.NODE_PHASE).map((phase) => ({
                value: (models.NODE_PHASE as any)[phase],
                label: (models.NODE_PHASE as any)[phase],
            })),
            selectedValues: this.phases,
            selectionChanged: (phases) => {
                const query = phases.length > 0 ? '?' + phases.map((phase) => `phase=${phase}`).join('&') : '';
                this.appContext.router.history.push(uiUrl(`workflows${query}`));
            },
        };
        return (
            <Page title='Workflows' toolbar={{filter, breadcrumbs: [{ title: 'Workflows', path: uiUrl('workflows') }]}}>
                <div className='argo-container'>
                    <div className='stream'>
                        <DataLoader
                                input={this.phases}
                                load={(phases) => {
                                    return Observable.fromPromise(services.workflows.list(phases)).flatMap((workflows) =>
                                        Observable.merge(
                                            Observable.from([workflows]),
                                            services.workflows.watch(phases).map((workflowChange) => {
                                                const index = workflows.findIndex((item) => item.metadata.name === workflowChange.object.metadata.name);
                                                if (index > -1 && workflowChange.object.metadata.resourceVersion === workflows[index].metadata.resourceVersion) {
                                                    return {workflows, updated: false};
                                                }
                                                switch (workflowChange.type) {
                                                    case 'DELETED':
                                                        if (index > -1) {
                                                            workflows.splice(index, 1);
                                                        }
                                                        break;
                                                    default:
                                                        if (index > -1) {
                                                            workflows[index] = workflowChange.object;
                                                        } else {
                                                            workflows.unshift(workflowChange.object);
                                                        }
                                                        break;
                                                }
                                                return {workflows, updated: true};
                                            }).filter((item) => item.updated).map((item) => item.workflows)),
                                    );
                                }}
                                loadingRenderer={() => <MockupList height={150} marginTop={30}/>}>
                            {(workflows: models.Workflow[]) => workflows.map((workflow) => (
                                <div key={workflow.metadata.name}>
                                    <Link to={uiUrl(`workflows/${workflow.metadata.namespace}/${workflow.metadata.name}`)}>
                                    <WorkflowListItem workflow={workflow}/>
                                    </Link>
                                </div>
                            ))}
                        </DataLoader>
                    </div>
                </div>
            </Page>
        );
    }

    private get appContext(): AppContext {
        return this.context as AppContext;
    }
}
