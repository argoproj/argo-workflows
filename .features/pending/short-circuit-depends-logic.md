Description: Short-circuit evaluation in enhanced depends logic
Authors: [Prashanth Chaitanya](https://github.com/prashanthjos)
Component: General
Issues: 15244

The enhanced `depends` logic now supports short-circuit evaluation, allowing DAG tasks to proceed earlier when the outcome of a depends expression is already determined by a subset of completed dependencies.

For example, with `depends: "A.Succeeded || B.Succeeded"`, if task A has already succeeded, the dependent task will proceed immediately without waiting for task B to complete.

This is a transparent performance optimization.
The syntax and semantics of depends expressions remain unchanged — tasks will reach the same final state, but may do so faster when short-circuit evaluation applies.
