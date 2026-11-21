import {History} from 'history';

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
