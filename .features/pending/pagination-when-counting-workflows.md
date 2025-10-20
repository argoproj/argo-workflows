Component: UI
Issues: 13948
Description: Optimize pagination performance when counting workflows in archive.
Author: [Shuangkun Tian](https://github.com/shuangkun)

When querying archived workflows with pagination, the system now uses more efficient methods to check if there are more items available. Instead of performing expensive full table scans, the new implementation uses LIMIT queries to check if there are items beyond the current offset+limit, significantly improving performance for large datasets.