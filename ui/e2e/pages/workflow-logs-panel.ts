import {Locator, Page} from '@playwright/test';

// Page object for the Logs side panel (ui/src/workflows/components/workflow-logs-viewer),
// opened from the details page toolbar.
export class WorkflowLogsPanel {
    readonly container: Locator;
    readonly heading: Locator;
    readonly viewer: Locator;

    constructor(page: Page) {
        this.container = page.locator('.workflow-logs-viewer');
        this.heading = this.container.getByRole('heading', {name: 'Logs'});
        // argo-ui's LogsViewer, mounted once the first log entry (or end of
        // stream) replaces the "Waiting for data..." placeholder.
        this.viewer = this.container.locator('.logs-viewer');
    }

    /**
     * The text currently shown in the log terminal, one line per row.
     *
     * argo-ui's LogsViewer renders through xterm, which paints to a canvas, so
     * the text is not in the DOM for a locator to read. It is in xterm's line
     * buffer, and LogsViewer keeps its Terminal on the component instance, which
     * React exposes from the root element through the fiber tree. This reads
     * what a user would get by selecting all and copying, and is polled because
     * xterm writes asynchronously.
     */
    async text(): Promise<string> {
        return this.viewer.evaluate(el => {
            const fiberKey = Object.keys(el).find(key => key.startsWith('__reactFiber$'));
            let fiber = fiberKey ? (el as any)[fiberKey] : undefined;
            while (fiber && !fiber.stateNode?.terminal) {
                fiber = fiber.return;
            }
            if (!fiber) {
                throw new Error('no LogsViewer component (with a terminal) above .logs-viewer');
            }
            const buffer = fiber.stateNode.terminal.buffer.active;
            const lines: string[] = [];
            for (let y = 0; y < buffer.length; y++) {
                lines.push(buffer.getLine(y)?.translateToString(true) ?? '');
            }
            return lines.join('\n').trimEnd();
        });
    }
}
