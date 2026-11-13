Description: Configurable list limit on the cron workflow details page
Author: [Chris](https://github.com/ChrisJr404)
Component: UI
Issues: 14065

The historical-runs list on the cron workflow details page now supports configurable pagination instead of a hard-coded 50-row cap. Use the `limit` query parameter (e.g. `?limit=100`) or the per-page selector in the new pagination footer to scroll back through older runs.
