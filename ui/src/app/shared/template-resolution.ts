import * as models from '../../models';
import {execSpec, ResourceScope, WorkflowStep} from '../../models';

export function getResolvedTemplates(workflow: models.Workflow, node: models.NodeStatus): models.Template {
    let tmpTemplate = {
        template: node.templateName,
        templateRef: node.templateRef
    } as WorkflowStep;
    const scope = getTemplateScope(node);
    const referencedTemplates: models.Template[] = [];
    let resolvedTemplate: models.Template = null;
    const maxDepth = 10;
    for (let i = 1; i < maxDepth + 1; i++) {
        const templRef = resolveTemplateReference(scope.ResourceScope, scope.ResourceName, tmpTemplate, scope.CompatibilityMode);
        let tmpl = null;
        if (templRef.StorageNeeded) {
            tmpl = workflow.status.storedTemplates[templRef.StoredTemplateName];
        } else if (tmpTemplate.template) {
            tmpl = execSpec(workflow).templates.find(item => item.name === tmpTemplate.template);
        }
        if (!tmpl) {
            const name = templRef.StoredTemplateName || tmpTemplate.template;
            console.error(`StoredTemplate ${name} not found`);
            return undefined;
        }
        referencedTemplates.push(tmpl);
        if (!tmpl.template && !tmpl.templateRef) {
            break;
        }
        tmpTemplate = tmpl;
        if (i === maxDepth) {
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

// resolveTemplateReference resolves the stored template name of a given template holder on the template scope and determines
// if it should be stored
function resolveTemplateReference(
    callerScope: ResourceScope,
    resourceName: string,
    caller: WorkflowStep,
    compatibilityMode: boolean
): {StoredTemplateName: string; StorageNeeded: boolean} {
    if (caller.templateRef) {
        // We are calling an external WorkflowTemplate or ClusterWorkflowTemplate. Template storage is needed
        // We need to determine if we're calling a WorkflowTemplate or a ClusterWorkflowTemplate
        const referenceScope: ResourceScope = caller.templateRef.clusterScope ? 'cluster' : 'namespaced';
        let name = caller.templateRef.name + '/' + caller.templateRef.template;
        if (!compatibilityMode) {
            name = referenceScope + '/' + name;
        }
        return {StoredTemplateName: name, StorageNeeded: true};
    } else if (callerScope !== 'local') {
        // Either a WorkflowTemplate or a ClusterWorkflowTemplate is calling a template inside itself. Template storage is needed
        let name = resourceName + '/' + caller.template;
        if (!compatibilityMode) {
            name = callerScope + '/' + name;
        }
        return {StoredTemplateName: name, StorageNeeded: true};
    } else {
        // A Workflow is calling a template inside itself. Template storage is not needed
        return {StoredTemplateName: '', StorageNeeded: false};
    }
}

function getTemplateScope(nodeStatus: models.NodeStatus): {CompatibilityMode: boolean; ResourceScope: ResourceScope; ResourceName?: string} {
    // For compatibility: an empty TemplateScope is a local scope
    if (!nodeStatus.templateScope) {
        return {CompatibilityMode: true, ResourceScope: 'local'};
    }
    const split = nodeStatus.templateScope.split('/');
    // For compatibility: an unspecified ResourceScope in a TemplateScope is a namespaced scope
    if (split.length === 1) {
        return {CompatibilityMode: true, ResourceScope: 'namespaced', ResourceName: split[0]};
    }
    return {CompatibilityMode: false, ResourceScope: split[0] as ResourceScope, ResourceName: split[1]};
}
