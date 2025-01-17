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
