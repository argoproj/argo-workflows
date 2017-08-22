import { Task } from './task';

export interface BranchTasks {
    branch: string;
    repo: string;
    tasks: Task[];
}
