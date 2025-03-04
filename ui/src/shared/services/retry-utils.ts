/**
 * Utility function to retry an async operation with exponential backoff
 *
 * @param operation The async operation to retry
 * @param maxRetries Maximum number of retry attempts
 * @param baseDelay Base delay in milliseconds between retries (will be multiplied by 2^retryCount)
 * @param maxDelay Maximum delay in milliseconds
 * @returns The result of the operation
 * @throws The last error encountered if all retries fail
 */
export async function retryWithBackoff<T>(operation: () => Promise<T>, maxRetries: number = 3, baseDelay: number = 300, maxDelay: number = 3000): Promise<T> {
    let lastError: Error | null = null;

    for (let retryCount = 0; retryCount <= maxRetries; retryCount++) {
        try {
            return await operation();
        } catch (error) {
            console.warn(`Operation failed (attempt ${retryCount + 1}/${maxRetries + 1})`, error);
            lastError = error as Error;

            if (retryCount === maxRetries) {
                break;
            }

            // Calculate delay with exponential backoff
            const delay = Math.min(baseDelay * Math.pow(2, retryCount), maxDelay);

            // Add some randomness to prevent all clients retrying simultaneously
            const jitter = Math.random() * 100;

            // Wait before next retry
            await new Promise(resolve => setTimeout(resolve, delay + jitter));
        }
    }

    throw lastError;
}

/**
 * Utility function to retry an API call with a timeout
 *
 * @param apiCall The API call function to retry
 * @param timeout Timeout in milliseconds
 * @param maxRetries Maximum number of retry attempts
 * @returns The result of the API call
 */
export async function retryApiCall<T>(apiCall: () => Promise<T>, timeout: number = 10000, maxRetries: number = 2): Promise<T> {
    // Create a promise that rejects after the timeout
    const timeoutPromise = new Promise<never>((_, reject) => {
        setTimeout(() => reject(new Error('API call timed out')), timeout);
    });

    // Wrap the API call with retry logic
    const apiCallWithRetry = () => Promise.race([apiCall(), timeoutPromise]);

    return retryWithBackoff(apiCallWithRetry, maxRetries);
}
