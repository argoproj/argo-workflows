import {Page} from 'argo-ui';
import * as React from 'react';
import {RouteComponentProps} from 'react-router-dom';
import {WorkflowEventBinding} from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {BasePage} from '../../../shared/components/base-page';
import {ErrorNotice} from '../../../shared/components/error-notice';
import {GraphPanel} from '../../../shared/components/graph/graph-panel';
import {Graph} from '../../../shared/components/graph/types';
import {Loading} from '../../../shared/components/loading';
import {NamespaceFilter} from '../../../shared/components/namespace-filter';
import {services} from '../../../shared/services';
import {Utils} from '../../../shared/utils';

interface State {
    namespace: string;
    error?: Error;
    workflowEventBindings?: WorkflowEventBinding[];
}

export class WorkflowEventBindingsList extends BasePage<RouteComponentProps<any>, State> {
    private get namespace() {
        return this.state.namespace;
    }

    private set namespace(namespace: string) {
        this.fetch(namespace);
    }

    private get graph() {
        const g = new Graph();
        this.state.workflowEventBindings.forEach(web => {
            const eventId = web.metadata.uid + '/event';
            g.nodes.set(eventId, {label: web.spec.event.selector, type: 'event', icon: 'cloud'});
            if (web.spec.submit) {
                const templateName = web.spec.submit.workflowTemplateRef.name;
                g.nodes.set(templateName, {label: templateName, type: 'template', icon: 'window-maximize'});
                g.edges.set({v: eventId, w: templateName}, {});
            }
        });

        return g;
    }

    constructor(props: RouteComponentProps<any>, context: any) {
        super(props, context);
        this.state = {namespace: this.props.match.params.namespace || ''};
    }

    public componentDidMount() {
        this.fetch(this.namespace);
    }

    public render() {
        return (
            <Page
                title='Workflow Event Bindings'
                toolbar={{
                    tools: [<NamespaceFilter key='namespace-filter' value={this.namespace} onChange={namespace => (this.namespace = namespace)} />]
                }}>
                {this.state.error && <ErrorNotice error={this.state.error} />}
                {!this.state.workflowEventBindings ? (
                    <Loading />
                ) : (
                    <GraphPanel graph={this.graph} types={{event: true, template: true}} classNames={{'': true}} horizontal={true} />
                )}
            </Page>
        );
    }

    private saveHistory() {
        this.appContext.router.history.push(uiUrl(`workflow-event-bindings/${this.namespace}`));
        Utils.setCurrentNamespace(this.namespace);
    }

    private fetch(namespace: string) {
        services.event
            .listWorkflowEventBindings(namespace)
            .then(list =>
                this.setState(
                    {
                        workflowEventBindings: list.items || [],
                        namespace,
                        error: null
                    },
                    () => {
                        this.saveHistory();
                    }
                )
            )
            .catch(error => this.setState({error}));
    }
}
