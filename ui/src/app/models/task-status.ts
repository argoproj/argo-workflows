export enum TaskStatus {
    Skipped = -3,
    Cancelled = -2,
    Failed = -1,
    Success = 0,
    Waiting = 1,
    Running = 2,
    Canceling = 3,
    Init = 255,
}
