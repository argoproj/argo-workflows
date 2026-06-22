Component: General
Issues: 14112
Description: Add a configurable database connection timeout
Author: [HsiuChuanHsu](https://github.com/HsiuChuanHsu)

The persistence and synchronization database connections now apply a connection-establishment timeout.
Previously the controller could hang for the OS default (~75s) or indefinitely when the database was unreachable or half-open (TCP accepted but the handshake never completes).
Set `connectionTimeoutSeconds` under the database config (default 5 seconds) so the controller fails fast with a clear error and Kubernetes can restart it.
