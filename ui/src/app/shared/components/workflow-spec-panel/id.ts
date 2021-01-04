/*
 * Artifacts
 * OnExit
 * Parameters
 * StepGroup/{templateName}/{i}
 * Step/{templateName}/{i}/{j}
 * Task/{templateName}/{taskName}
 * Template/{templateName}
 * Workflow
 * WorkflowTemplateRef
 */

type Type = 'Workflow' | 'WorkflowTemplateRef' | 'Template' | 'StepGroup' | 'Step' | 'Task' | 'OnExit' | 'Parameters' | 'TemplateRef' | 'Artifacts';

export const workflowId: Type = 'Workflow';
export const parametersId: Type = 'Parameters';
export const artifactsId: Type = 'Artifacts';
export const onExitId: Type = 'OnExit';
export const workflowTemplateRefId: Type = 'WorkflowTemplateRef';

export const typeOf = (id: string): Type => id.split('/')[0] as Type;

export const idForStepGroup = (templateName: string, i: number) => 'StepGroup/' + templateName + '/' + i;

// TODO - we assume that template names are unique, but they may not be and that'll produce weird bugs
export const stepGroupOf = (id: string) => ({
    templateName: id.split('/')[1],
    i: parseInt(id.split('/')[2], 10)
});

export const idForSteps = (templateName: string, i: number, j: number) => 'Step/' + templateName + '/' + i + '/' + j;
export const stepOf = (id: string) => ({
    templateName: id.split('/')[1],
    i: parseInt(id.split('/')[2], 10),
    j: parseInt(id.split('/')[3], 10)
});

export const idForTemplate = (templateName: string) => 'Template/' + templateName;
export const templateOf = (id: string) => ({
    templateName: id.split('/')[1]
});

export const idForTask = (templateName: string, taskName: string) => 'Task/' + templateName + '/' + taskName;
export const taskOf = (id: string) => ({
    templateName: id.split('/')[1],
    taskName: id.split('/')[2]
});

export const idForTemplateRef = (templateName: string, template: string) => 'TemplateRef/' + templateName + '/' + template;
