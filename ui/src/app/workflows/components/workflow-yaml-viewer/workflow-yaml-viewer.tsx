import { SlideContents, Utils as ArgoUtils } from 'argo-ui';
import * as React from 'react';
import * as yaml from 'yamljs';

import * as models from '../../../../models';
import { Utils } from '../../../shared/utils';

require('./workflow-yaml-viewer.scss');

export interface WorkflowYamlViewerProps {
    workflow: models.Workflow; selectedNode: models.NodeStatus;
}

export class WorkflowYamlViewer extends React.Component<WorkflowYamlViewerProps> {

    private container: HTMLElement;

    public componentDidUpdate() {
        this.scrollToHighlightedStep();
    }

    public componentDidMount() {
        this.scrollToHighlightedStep();
    }

    public render() {
        const contents: JSX.Element[] = [];
        if (this.props.selectedNode) {
            const parentNode = this.props.workflow.status.nodes[this.props.selectedNode.boundaryID];
            if (parentNode) {
                const parentTemplate = Utils.getResolvedTemplates(this.props.workflow, parentNode);

                let nodeName = '';
                if (this.props.selectedNode) {
                    nodeName = this.normalizeNodeName(this.props.selectedNode.displayName || this.props.selectedNode.name);
                }
                let parentTemplateStr = yaml.stringify(parentTemplate, 4, 1);
                if (nodeName) {
                    parentTemplateStr = this.highlightStep(parentTemplate, nodeName, parentTemplateStr);
                }
                contents.push(
                <div className='workflow-yaml-section'>
                    <h4>Parent Node</h4>
                    <div dangerouslySetInnerHTML={{__html:  this.addCounterToDisplayedFiles(parentTemplateStr)}} />
                </div>,
                );
            }

            const template = Utils.getResolvedTemplates(this.props.workflow, this.props.selectedNode);
            const templateStr = yaml.stringify(template, 4, 1);
            contents.push(
                <div className='workflow-yaml-section'>
                    <h4>Current Node</h4>
                    <div dangerouslySetInnerHTML={{__html:  this.addCounterToDisplayedFiles(templateStr)}} />
                </div>,
                );
        }
        const templates = this.props.workflow.spec.templates;
        if (templates && Object.keys(templates).length) {
            const templatesStr = yaml.stringify(templates, 4, 1);
            contents.push((
                <SlideContents
                    title={'Templates'}
                    contents={<div dangerouslySetInnerHTML={{__html: this.addCounterToDisplayedFiles(templatesStr)}} />}
                    className='workflow-yaml-section'
                />
                ));
        }
        const storedTemplates = this.props.workflow.status.storedTemplates;
        if (storedTemplates && Object.keys(storedTemplates).length) {
            const storedTemplatesStr = yaml.stringify(storedTemplates, 4, 1);
            contents.push((
            <SlideContents
                title={'Stored Templates'}
                contents={<div dangerouslySetInnerHTML={{__html: this.addCounterToDisplayedFiles(storedTemplatesStr)}} />}
                className='workflow-yaml-section'
            />
            ));
        }

        return (
            <div className='workflow-yaml-viewer' ref={(container) => this.container = container}>
                {contents}
            </div>
        );
    }

    private addCounterToDisplayedFiles(multilineString: string): string {
        const newMultilineStringWithCounters: string[] = ['<ol>'];
        multilineString
            .split('\n')
            .forEach((item) => {
                if (item !== '') {
                    if (item.indexOf('<span>') !== -1) {
                        item = item.match(/^<span>\s*/)[0] + item.substr(6);
                        item = `<li class='highlight'>${item}</li>`;
                    } else {
                        item = item.match(/^\s*/)[0] + item;
                        // special treatment to beautify resource templates
                        if (item.replace(/\s+/g, '').substr(0, 8) === 'manifest') {
                            this.formatManifest(item, newMultilineStringWithCounters);
                            return;
                        }
                        item = `<li>${item}</li>`;
                    }
                }
                newMultilineStringWithCounters.push(item);
            });
        newMultilineStringWithCounters.push('</ol>');
        return newMultilineStringWithCounters.join('\n');
    }

    private highlightStep(template: models.Template, highlightedStepName: string, yamlString: string) {
        let firstLineStepToHighlight: string = null;
        let lastLineStepToHighlight: string = null;
        const steps: (models.WorkflowStep | models.DAGTask)[] = template.dag && template.dag.tasks || (template.steps || []).reduce((first, second) => first.concat(second), []);
        const step = steps.find((item) => item.name === highlightedStepName);
        if (step) {
            const stepLines = yaml
                .stringify(step, 1, 1)
                .split('\n');
            firstLineStepToHighlight = `name: ${highlightedStepName}`;
            lastLineStepToHighlight = stepLines[stepLines.length - 2];
        }

        if (firstLineStepToHighlight && lastLineStepToHighlight) {
            let newYamlString = '';
            let isLinePartOfStepToHighlight = false;

            yamlString
                .split('\n')
                .forEach((line: string, index) => {
                    if (line.indexOf(firstLineStepToHighlight) !== -1) {
                        isLinePartOfStepToHighlight = true;
                    }
                    if (isLinePartOfStepToHighlight) {
                        newYamlString = `${newYamlString}<span>${line}</span>\n`;
                        if (line.indexOf(lastLineStepToHighlight) > -1) {
                            isLinePartOfStepToHighlight = false;
                        }
                    } else {
                        newYamlString = `${newYamlString}${line}\n`;
                    }
                });

            yamlString = newYamlString;
        }
        return yamlString;
    }

    private normalizeNodeName(name: string) {
        const parts = name.replace(/([(][^)]*[)])/g, '').split('.');
        return parts[parts.length - 1];
    }

    private scrollToHighlightedStep() {
        if (this.props.selectedNode) {
            setTimeout(() => {
                const viewerHighlight = this.container.querySelector('li.highlight') as HTMLElement;
                if (viewerHighlight) {
                    const parent = ArgoUtils.getScrollParent(viewerHighlight);
                    ArgoUtils.scrollTo(
                        parent, viewerHighlight.offsetTop + parent.scrollTop - window.pageYOffset - parent.clientHeight / 2);
                }
            });
        }
    }

    private formatManifest(item: string, newMultilineStringWithCounters: string[]) {
        const index = item.indexOf('manifest:');
        item = item.substr(0, index + 10) + '\\n' + item.substr(index + 10);
        item = item.replace(/"/, '');
        item = item.replace(/\\"/g, '"');
        newMultilineStringWithCounters.push(`<li>${item.substr(0, index)}manifest: |`);
        item.split('\\n').slice(1).slice(0, -1).forEach((line) => {
            newMultilineStringWithCounters.push(`<li>${item.substr(0, index)}  ${line}</li>`);
        });
    }
}
