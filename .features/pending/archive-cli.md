Description: Allow archive cli commands to use workflow name instead of uid.
Author: [Isitha Subasinghe](https://github.com/isubasinghe)
Component: CLI
Issues: 15199 

This change allows for `archive` related cli commands to use the workflow name
instead of relying upon the uid. This is explicitly a user experience related improvement.
Note that if your name itself is a uid, you will have to manually force to fetch via uid or name, see the documentation for more detail.
