Component: General
Issues: 14069
Description: Name filter parameter for prefix/contains/exact search in `/archived-workflows`
Author: [Armin Friedl](https://github.com/arminfriedl)

A new `nameFilter` parameter was added to the `GET
/archived-workflows` endpoint. The filter works analogous to the one
in `GET /workflows`. It allows to specify how a search for
`?listOptions.fieldSelector=metadata.name=<search-string>` in these
endpoints should be interpreted. Possible values are `Prefix`,
`Contains` and `Exact`. The `metadata.name` field is matched
accordingly against the value for `<search-string>`.
