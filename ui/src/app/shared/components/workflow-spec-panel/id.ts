/*
 * Workflow/
 * WorkflowTemplateRef/
 * Template/{templateName}
 * StepGroup/{templateName},{i}
 * Step/{templateName},{i},{j}
 * Task/{templateName},{taskName}
 * OnExit/
 * Parameters/
 * Artifacts
 */

export type Type = 'Workflow' | 'WorkflowTemplateRef' | 'Template' | 'StepGroup' | 'Step' | 'Task' | 'OnExit' | 'Parameters' | 'TemplateRef' | 'Artifacts';

export const ID = {
    join: (type: Type, name = '') => type + '/' + name,
    split: (id: string) => {
        const parts = id.split('/');
        const type: Type = parts[0] as Type;
        const name = parts[1];
        return {type, name};
    }
};
