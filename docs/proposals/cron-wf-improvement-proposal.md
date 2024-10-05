# Proposal for Cron Workflows improvements

## Introduction

Currently, CronWorkflows are a great resource if we want to run recurring tasks to infinity. However, it is missing the ability to customize it, for example define how many times a workflow should run or how to handle multiple failures. I believe argo workflows would benefit of having more configuration options for cron workflows, to allow to change its behavior based on the result of its child’s success or failures. Below I present my thoughts on how we could improve them, but also some questions and concerns on how to properly do it.

## Proposal

This proposal discusses the viability of adding 2 more fields into the cron workflow configuration:

```yaml
RunStrategy:
 maxSuccess:
 maxFailures:
```

`maxSuccess` - defines how many child workflows must have success before suspending the workflow schedule

`maxFailures` - defines how many child workflows must fail before suspending the workflow scheduling. This may contain `Failed` workflows, `Errored` workflows or spec errors.

For example, if we want to run a workflow just once, we could just set:

```yaml
RunStrategy:
 maxSuccess: 1
```

This configuration will make sure the controller will keep scheduling workflows until one of them finishes with success.

As another example, if we want to stop scheduling workflows when they keep failing, we could configure the CronWorkflow with:

```yaml
RunStrategy:
 maxFailures: 2
```

This config will stop scheduling workflows if fails twice.

## Total vs consecutive

One aspect that needs to be discussed is whether these configurations apply to the entire life of a cron Workflow or just in consecutive schedules. For example, if we configure a workflow to stop scheduling after 2 failures, I think it makes sense to have this applied when it fails twice consecutively. Otherwise, we can have 2 outages in different periods which will suspend the workflow. On the other hand, when configuring a workflow to run twice with success, it would make more sense to have it execute with success regardless of whether it is a consecutive success or not. If we have an outage after the first workflow succeeds, which translates into failed workflows, it should need to execute with success only once. So I think it would make sense to have:

- maxFailures - maximum number of **consecutive failures** before stopping the scheduling of a workflow

- maxSuccess - maximum number of workflows with success.

## How to store state

Since we need to control how many child workflows had success/failure we must store state. With this some questions arise:

- Should we just store it through the lifetime of the controller or should we store it to a database?

    - Probably only makes sense if we can backup the state somewhere (like a BD). However, I don't have enough knowledge about workflow's architecture to tell how good of an idea this is.
- If a CronWorkflow gets re-applied, does it maintain or reset the number of success/failures?

    - I guess it should reset since a configuration change should be seen as a new start.

## How to stop the workflow

Once the configured number of failures or successes is reached, it is necessary to stop the workflow scheduling.
I believe we have 3 options:

- Delete the workflow: In my opinion, this is the worst option and goes against gitops principles.
- Suspend it (set suspend=true): the workflow spec is changed to have the workflow suspended. I may be wrong but this conflicts with gitops as well.
- Stop scheduling it: The workflow spec is the same. The controller needs to check if the max number of runs was already attained and skip scheduling if it did.

Option 3 seems to be the only possibility. After reaching the max configured executions, the cron workflow would exist forever but never scheduled. Maybe we could add a new status field, like `Inactive` and have something the UI to show it?

## How to handle suspended workflows

One possible case that comes to mind is a long outage where all workflows are failing. For example, imagine a workflow that needs to download a file from some storage and for some reason that storage is down. Workflows will keep getting scheduled but they are going to fail. If they fail the number of configured `maxFailures`, the workflows gets stopped forever. Once the storage is back up, how can the user enable the workflow again?

- Manually re-create the workflow: could be an issue if the user has multiple cron workflows
- Instead of stopping the workflow scheduling, introduce a back-off period as suggested by [#7291](https://github.com/argoproj/argo-workflows/issues/7291). Or maybe allow both configurations.

I believe option 2 would allow the user to select if they want to stop scheduling or not. If they do, when cron workflows are wrongfully halted, they will need to manually start them again. If they don't, Argo will only introduce a back-off period between schedules to avoid rescheduling workflows that are just going to fail. Spec could look something like:

```yaml
RunStrategy:
 maxSuccess:
 maxFailures:
  value: # this would be optional
  back-off:
   enabled: true
   factor: 2
```

With this configuration the user could configure 3 behaviors:

1. set `value` if they wanted to stop scheduling a workflow after a maximum number of consecutive failures.
2. set `value` and `back-off` if they wanted to stop scheduling a workflow after a maximum number of consecutive failures but with a back-off period between each failure
3. set `back-off` if they want a back-off period between each failure but they never want to stop the workflow scheduling.

## Wrap up

I believe this feature would enhance the cron workflows to allow more specific use cases that are commonly requested by the community, such as running a workflow only once. This proposal raises some concerns on how to properly implement it and I would like to know the maintainers/contributors opinion on these 4 topics, but also some other issues that I couldn't think of.

## Resources

- This discussion was prompted by [#10620](https://github.com/argoproj/argo-workflows/issues/10620)
- A first approach to this problem was discussed in [5659](https://github.com/argoproj/argo-workflows/issues/5659)
- A draft PR to implement the first approach [#5662](https://github.com/argoproj/argo-workflows/pull/5662)
