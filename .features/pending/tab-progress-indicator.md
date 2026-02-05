Description: Show workflow status in browser tab favicon and title
Authors: [André Ahlert](https://github.com/andreahlert)
Component: UI
Issues: 15226

When viewing a workflow's details page, the browser tab now displays the workflow's current status.
This makes it easy to monitor long-running workflows while working in other tabs.

The feature includes:
* A colored status indicator dot overlaid on the favicon (red for failed, yellow/orange for running, green for succeeded)
* The document title is updated to show the workflow phase and name, e.g. `[Running] my-workflow - Argo`

This is a common pattern in CI/CD tools like GitHub Actions and GitLab CI.
