import * as PropTypes from 'prop-types';
import * as React from 'react';
import {Link, RouteComponentProps} from 'react-router-dom';
import {Observable} from 'rxjs';

import {Autocomplete, DataLoader, MockupList, Page, SlidingPanel, TopBarFilter} from 'argo-ui';
import * as models from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {AppContext, Consumer} from '../../../shared/context';
import {services} from '../../../shared/services';

import {WorkflowListItem} from '..';
import {Query} from '../../../shared/components/query';
import { YamlEditor } from '../../../shared/components/yaml-editor/yaml-editor';

require('./workflows-list.scss');

function tryJsonParse(input: string) {
    try {
        return (input && JSON.parse(input)) || null;
    } catch {
        return null;
    }
}

export class WorkflowsList extends React.Component<RouteComponentProps<any>> {
    public static contextTypes = {
        router: PropTypes.object,
        apis: PropTypes.object
    };

    private get phases() {
        return new URLSearchParams(this.props.location.search).getAll('phase');
    }

    private get wfInput() {
        const query = new URLSearchParams(this.props.location.search);
        return tryJsonParse(query.get('new'));
    }

    public render() {
        const filter: TopBarFilter<string> = {
            items: Object.keys(models.NODE_PHASE).map(phase => ({
                value: (models.NODE_PHASE as any)[phase],
                label: (models.NODE_PHASE as any)[phase]
            })),
            selectedValues: this.phases,
            selectionChanged: phases => {
                const query = phases.length > 0 ? '?' + phases.map(phase => `phase=${phase}`).join('&') : '';
                this.appContext.router.history.push(uiUrl(`workflows${query}`));
            }
        };
        return (
            <Consumer>
                {ctx => (
                    <Page
                        title='Workflows'
                        toolbar={{
                            filter,
                            breadcrumbs: [{title: 'Workflows', path: uiUrl('workflows')}],
                            actionMenu: {
                                items: [
                                    {
                                        title: 'Submit New Workflow',
                                        iconClassName: 'fa fa-plus',
                                        action: () => ctx.navigation.goto('.', {new: '{}'})
                                    }
                                ]
                            }
                        }}>
                        <div className='workflows-list'>
                            <DataLoader
                                input={this.phases}
                                load={phases => {
                                    // TODO(simon): Remove hardwired 'argo' namespace
                                    return Observable.fromPromise(services.workflows.list(phases, 'argo')).flatMap(workflows =>
                                        Observable.merge(
                                            Observable.from([workflows]),
                                            services.workflows
                                                .watch(phases)
                                                .map(workflowChange => {
                                                    const index = workflows.findIndex(item => item.metadata.name === workflowChange.object.metadata.name);
                                                    if (index > -1 && workflowChange.object.metadata.resourceVersion === workflows[index].metadata.resourceVersion) {
                                                        return {workflows, updated: false};
                                                    }
                                                    if (workflowChange.type === 'DELETED') {
                                                        if (index > -1) {
                                                            workflows.splice(index, 1);
                                                        }
                                                    } else {
                                                        if (index > -1) {
                                                            workflows[index] = workflowChange.object;
                                                        } else {
                                                            workflows.unshift(workflowChange.object);
                                                        }
                                                    }
                                                    return {workflows, updated: true};
                                                })
                                                .filter(item => item.updated)
                                                .map(item => item.workflows)
                                        )
                                    );
                                }}
                                loadingRenderer={() => <MockupList height={150} marginTop={30} />}>
                                {(workflows: models.Workflow[]) => (
                                    <div className='row'>
                                        <div className='columns small-12 xxlarge-2'>
                                            <Query>
                                                {q => (
                                                    <div className='workflows-list__search'>
                                                        <i className='fa fa-search' />
                                                        {q.get('search') && (
                                                            <i
                                                                className='fa fa-times'
                                                                onClick={() => {
                                                                    ctx.navigation.goto('.', {search: null}, {replace: true});
                                                                }}
                                                            />
                                                        )}
                                                        <Autocomplete
                                                            filterSuggestions={true}
                                                            renderInput={inputProps => (
                                                                <input
                                                                    {...inputProps}
                                                                    onFocus={e => {
                                                                        e.target.select();
                                                                        if (inputProps.onFocus) {
                                                                            inputProps.onFocus(e);
                                                                        }
                                                                    }}
                                                                    className='argo-field'
                                                                />
                                                            )}
                                                            renderItem={item => (
                                                                <React.Fragment>
                                                                    <i className='icon argo-icon-workflow' /> {item.label}
                                                                </React.Fragment>
                                                            )}
                                                            onSelect={val => {
                                                                ctx.navigation.goto(`./${val}`);
                                                            }}
                                                            onChange={e => {
                                                                ctx.navigation.goto('.', {search: e.target.value}, {replace: true});
                                                            }}
                                                            value={q.get('search') || ''}
                                                            items={workflows.map(wf => wf.metadata.namespace + '/' + wf.metadata.name)}
                                                        />
                                                    </div>
                                                )}
                                            </Query>
                                        </div>

                                        <div className='stream'>
                                            <div className='columns small-12 xxlarge-10'>
                                                {workflows.map(workflow => (
                                                    <div key={workflow.metadata.name}>
                                                        <Link to={uiUrl(`workflows/${workflow.metadata.namespace}/${workflow.metadata.name}`)}>
                                                            <WorkflowListItem workflow={workflow} history={false} />
                                                        </Link>
                                                    </div>
                                                ))}
                                            </div>
                                        </div>
                                    </div>
                                )}
                            </DataLoader>
                        </div>
                        <SlidingPanel
                            isShown={!!this.wfInput}
                            onClose={() => ctx.navigation.goto('.', {new: null})}
                        >
                            Submit Workflow
                            <YamlEditor
                                minHeight={800}
                                initialEditMode={true}
                                onCancel={() => alert('Cancel')}
                                onSave={rawWf => {
                                    // TODO(simon): Remove hardwired 'argo' namespace
                                    services.workflows.create(JSON.parse(rawWf), 'argo').then()
                                        .then(wf => ctx.navigation.goto(`/workflows/${wf.metadata.namespace}/${wf.metadata.name}`));
                                }}
                            />
                        </SlidingPanel>
                    </Page>
                )}
            </Consumer>
        );
    }

    private get appContext(): AppContext {
        return this.context as AppContext;
    }
}
