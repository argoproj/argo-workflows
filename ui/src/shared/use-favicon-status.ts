import {useEffect, useRef} from 'react';

import {NODE_PHASE, NodePhase} from './models';

const FAVICON_SIZE = 32;
const DEFAULT_FAVICON = 'assets/favicon/favicon-32x32.png';

/**
 * Returns the color associated with a workflow phase
 */
function getPhaseColor(phase: NodePhase): string {
    switch (phase) {
        case NODE_PHASE.ERROR:
        case NODE_PHASE.FAILED:
            return '#E96D76'; // red
        case NODE_PHASE.RUNNING:
        case NODE_PHASE.PENDING:
            return '#F5A623'; // yellow/orange for in-progress (more visible than blue)
        case NODE_PHASE.SUCCEEDED:
            return '#18BE94'; // green
        default:
            return null; // use default favicon
    }
}

/**
 * Creates a favicon with a colored status indicator dot
 */
function createStatusFavicon(color: string): Promise<string> {
    return new Promise((resolve, reject) => {
        const canvas = document.createElement('canvas');
        canvas.width = FAVICON_SIZE;
        canvas.height = FAVICON_SIZE;
        const ctx = canvas.getContext('2d');

        if (!ctx) {
            reject(new Error('Could not get canvas context'));
            return;
        }

        const img = new Image();
        img.crossOrigin = 'anonymous';
        img.onload = () => {
            // Draw original favicon
            ctx.drawImage(img, 0, 0, FAVICON_SIZE, FAVICON_SIZE);

            // Draw colored circle indicator in bottom-right corner
            const circleRadius = 8;
            const circleX = FAVICON_SIZE - circleRadius - 2;
            const circleY = FAVICON_SIZE - circleRadius - 2;

            // White border
            ctx.beginPath();
            ctx.arc(circleX, circleY, circleRadius + 1, 0, 2 * Math.PI);
            ctx.fillStyle = '#FFFFFF';
            ctx.fill();

            // Colored circle
            ctx.beginPath();
            ctx.arc(circleX, circleY, circleRadius, 0, 2 * Math.PI);
            ctx.fillStyle = color;
            ctx.fill();

            resolve(canvas.toDataURL('image/png'));
        };
        img.onerror = () => {
            reject(new Error('Failed to load favicon image'));
        };
        img.src = DEFAULT_FAVICON;
    });
}

/**
 * Updates the favicon link element
 */
function setFavicon(href: string) {
    let link: HTMLLinkElement = document.querySelector("link[rel*='icon']");
    if (!link) {
        link = document.createElement('link');
        link.rel = 'icon';
        document.head.appendChild(link);
    }
    link.type = 'image/png';
    link.href = href;
}

/**
 * Hook that updates the browser tab favicon based on workflow phase.
 * Also updates the document title to show the phase.
 *
 * @param phase - The current workflow phase
 * @param workflowName - The workflow name to show in title
 */
export function useFaviconStatus(phase: NodePhase, workflowName?: string) {
    const originalTitle = useRef<string>(document.title);
    const originalFavicon = useRef<string>(DEFAULT_FAVICON);

    useEffect(() => {
        // Store original values on mount
        originalTitle.current = document.title;
        const existingFavicon = document.querySelector("link[rel*='icon']") as HTMLLinkElement;
        if (existingFavicon) {
            originalFavicon.current = existingFavicon.href;
        }

        return () => {
            // Restore original values on unmount
            document.title = 'Argo';
            setFavicon(originalFavicon.current);
        };
    }, []);

    useEffect(() => {
        if (!phase) {
            return;
        }

        const color = getPhaseColor(phase);

        // Update document title
        if (workflowName) {
            const phaseLabel = phase === NODE_PHASE.RUNNING ? 'Running' : phase === NODE_PHASE.SUCCEEDED ? 'Succeeded' : phase === NODE_PHASE.FAILED ? 'Failed' : phase;
            document.title = `[${phaseLabel}] ${workflowName} - Argo`;
        }

        // Update favicon
        if (color) {
            createStatusFavicon(color)
                .then(setFavicon)
                .catch(err => {
                    console.warn('Failed to update favicon:', err);
                });
        } else {
            setFavicon(DEFAULT_FAVICON);
        }
    }, [phase, workflowName]);
}
