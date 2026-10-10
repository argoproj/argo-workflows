export type HttpError = Error & {response?: {text?: string; body?: {message?: string}}};

// Returns a readable message for an error.
// If the error was a HTTP error (i.e. from super-agent), rather than just an unhelpful "Forbidden",
// it will include any message in the body.
export function getErrorMessage(error: HttpError): string {
    const bodyMessage = error?.response?.body?.message;
    if (bodyMessage) {
        // without a status text, which HTTP/2 never has, super-agent uses the raw response text as the error message
        const isRawBody = error.message === error.response.text;
        return error.message && !isRawBody ? `${error.message}: ${bodyMessage}` : bodyMessage;
    }
    return error?.message || 'Unknown error. Open your browser error console for more information.';
}
