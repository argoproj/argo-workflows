export type HttpError = Error & {response?: {body?: {message?: string}}};

// Returns a readable message for an error.
// If the error was a HTTP error (i.e. from super-agent), rather than just an unhelpful "Forbidden",
// it will include any message in the body.
export function getErrorMessage(error: HttpError): string {
    const message = error?.message || 'Unknown error. Open your browser error console for more information.';
    const bodyMessage = error?.response?.body?.message;
    return bodyMessage ? `${message}: ${bodyMessage}` : message;
}
