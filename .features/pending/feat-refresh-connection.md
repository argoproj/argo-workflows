Component: General
Issues: 15011
Description: Reconnect and retry queries
Author: [Isitha Subasinghe](https://github.com/isubasinghe)

Queries against the database are now retried where a network connection issue was the cause of failure, this
is done through reconnecting first.
