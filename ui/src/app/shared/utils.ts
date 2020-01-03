import {Observable} from 'rxjs';
import * as models from '../../models';
import {NODE_PHASE} from '../../models';

export const Utils = {
    statusIconClasses(status: string): string {
        let classes = [];
        switch (status) {
            case NODE_PHASE.ERROR:
            case NODE_PHASE.FAILED:
                classes = ['fa-times-circle', 'status-icon--failed'];
                break;
            case NODE_PHASE.SUCCEEDED:
                classes = ['fa-check-circle', 'status-icon--success'];
                break;
            case NODE_PHASE.RUNNING:
                classes = ['fa-circle-notch', 'status-icon--running', 'status-icon--spin'];
                break;
            case NODE_PHASE.PENDING:
                classes = ['fa-clock', 'status-icon--pending', 'status-icon--slow-spin'];
                break;
            default:
                classes = ['fa-clock', 'status-icon--init'];
                break;
        }
        return classes.join(' ');
    },

    shortNodeName(node: {name: string; displayName: string}): string {
        return node.displayName || node.name;
    },

    toObservable<T>(val: T | Observable<T> | Promise<T>): Observable<T> {
        const observable = val as Observable<T>;
        if (observable && observable.subscribe && observable.catch) {
            return observable as Observable<T>;
        }
        return Observable.from([val as T]);
    },

    getResolvedTemplates(workflow: models.Workflow, node: models.NodeStatus): models.Template {
        let tmpTemplate = {
            template: node.templateName,
            templateRef: node.templateRef
        };
        let scope = node.templateScope;
        const referencedTemplates: models.Template[] = [];
        let resolvedTemplate: models.Template;
        const maxDepth = 10;
        for (let i = 1; i < maxDepth + 1; i++) {
            let storedTemplateName = '';
            if (tmpTemplate.templateRef) {
                storedTemplateName = `${tmpTemplate.templateRef.name}/${tmpTemplate.templateRef.template}`;
                scope = tmpTemplate.templateRef.name;
            } else {
                storedTemplateName = `${scope}/${tmpTemplate.template}`;
            }
            let tmpl = null;
            if (scope && storedTemplateName) {
                tmpl = workflow.status.storedTemplates[storedTemplateName];
            } else if (tmpTemplate.template) {
                tmpl = workflow.spec.templates.find(item => item.name === tmpTemplate.template);
            }
            if (!tmpl) {
                // tslint:disable-next-line: no-console
                console.error(`StoredTemplate ${storedTemplateName} not found`);
                return undefined;
            }
            referencedTemplates.push(tmpl);
            if (!tmpl.template && !tmpl.templateRef) {
                break;
            }
            tmpTemplate = tmpl;
            if (i === maxDepth) {
                // tslint:disable-next-line: no-console
                console.error(`Template reference too deep`);
                return undefined;
            }
        }
        referencedTemplates.reverse().forEach(tmpl => {
            tmpl = Object.assign({}, tmpl);
            delete tmpl.template;
            delete tmpl.templateRef;
            resolvedTemplate = Object.assign({}, resolvedTemplate, tmpl);
        });
        return resolvedTemplate;
    }
};
