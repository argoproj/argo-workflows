import {History} from 'history';

import {Parameter} from './models';

function extractKey(inputString: string): string | null {
    // Use regular expression to match the key within square brackets
    const match = inputString.match(/^parameters\[(.*?)\]$/);

    // If a match is found, return the captured key
    if (match) {
        return match[1];
    }

    // If no match is found, return null or an empty string
    return null; // Or return '';
}
/**
 * Returns the workflow parameters from the query parameters.
 */
export function getWorkflowParametersFromQuery(history: History): {[key: string]: string} {
    const queryParams = new URLSearchParams(history.location.search);

    const parameters: {[key: string]: string} = {};
    for (const [key, value] of queryParams.entries()) {
        const q = extractKey(key);
        if (q) {
            parameters[q] = value;
        }
    }

    return parameters;
}

const defaultSubmitLabels = ['submit-from-ui=true'];

/**
 * Returns true if the query parameter is used to prefill the submit workflow panel.
 */
function isSubmitPrefillKey(key: string): boolean {
    return extractKey(key) !== null || key === 'entrypoint' || key === 'labels';
}

/**
 * Returns the query parameters used to prefill the submit workflow panel
 * (`parameters[...]`, `entrypoint` and `labels`), or null if there are none.
 * Pages that rewrite their URL use this to keep those parameters until the panel reads them.
 */
export function getSubmitPrefillSearchParams(search: string): URLSearchParams | null {
    const prefill = new URLSearchParams();
    new URLSearchParams(search).forEach((value, key) => {
        if (isSubmitPrefillKey(key)) {
            prefill.set(key, value);
        }
    });
    return prefill.toString() ? prefill : null;
}

/**
 * Returns the entrypoint from the query parameters, or null if it is not set.
 */
export function getEntrypointFromQuery(history: History): string | null {
    return new URLSearchParams(history.location.search).get('entrypoint') || null;
}

/**
 * Returns the comma-separated labels from the query parameters, or the default submit labels if none are set.
 */
export function getLabelsFromQuery(history: History): string[] {
    const labels = (new URLSearchParams(history.location.search).get('labels') || '')
        .split(',')
        .map(label => label.trim())
        .filter(label => label);
    return labels.length > 0 ? labels : defaultSubmitLabels;
}

/**
 * Returns the input parameters of the entrypoint template, prefilled from `parameters[...]` in the query parameters.
 * Inputs with the same name as a workflow parameter are left out: the workflow parameter already carries that value,
 * and submitting both would let the template default override the workflow argument.
 * An empty query value (`parameters[foo]=`) is kept rather than replaced by the template value.
 * Only the query's own keys count, so an input named like an Object.prototype member (e.g. `constructor`) keeps its template value.
 */
export function getEntrypointParametersFromQuery(history: History, templateParameters: Parameter[], workflowParameters: Parameter[]): Parameter[] {
    const valuesFromQuery = getWorkflowParametersFromQuery(history);
    const workflowParameterNames = new Set(workflowParameters.map(param => param.name));
    return templateParameters
        .filter(param => !workflowParameterNames.has(param.name))
        .map(param => ({
            ...param,
            value: Object.prototype.hasOwnProperty.call(valuesFromQuery, param.name) ? valuesFromQuery[param.name] : param.value
        }));
}
