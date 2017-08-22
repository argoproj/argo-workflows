export const STATUS_FILTERS: { [key: string]: { title: string, statuses: string[], color: string } } = {
    TRANSIENT: { title: 'Transient', statuses: ['Waiting', 'Terminating', 'Stopping', 'Init'], color: 'running' },
    ACTIVE: { title: 'Active', statuses: ['Active'], color: 'success' },
    ERROR: { title: 'Error',  statuses: ['Error'], color: 'failed' },
    TERMINATED: { title: 'Terminated', statuses: ['Terminated'], color: 'running' },
    STOPPED: { title: 'Stopped', statuses: ['Stopped'], color: 'stopped' },
    UPGRADING: { title: 'Upgrading', statuses: ['Upgrading'], color: 'running' }
};
