export class CommitJob {
    name: string = '';
    status: number = 0;
    ax_time: number = 0;
    run_time: number = 0;
}

export class Commit {
    author: string = '';
    branch: string = '';
    branches: string[] = [];
    committer: string = '';
    date: number;
    description: string = '';
    project: string = '';
    repo: string = '';
    revision: string = '';
    jobs: CommitJob[] = [];
    jobs_wait: number = 0;
    jobs_init: number = 0;
    jobs_run: number = 0;
    jobs_fail: number = 0;
    jobs_success: number = 0;
}

export const CommitFieldNames = {
    description: 'description',
    author: 'author',
    committer: 'committer',
    branch: 'branch',
    repo: 'repo'
};
