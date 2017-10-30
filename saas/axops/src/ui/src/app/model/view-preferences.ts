import { Branch } from './branch';

export interface ViewPreferences {
    filterState: {
        repoBranch?: { repo: string, branch?: string },
        selectedBranch?: string;
        selectedRepo?: string;
        branches: 'all' | 'my';
    };
    favouriteBranches: Branch[];
    isIntroductionCompleted: boolean;
    playgroundTask: { jobId: string, projectId: string };
    mostRecentNotificationsViewTime: number;
    filterStateInPages: { [key: string]: any };
    firstJobFeedbackStatus: 'need-feedback' | 'feedback-submitted';
}
