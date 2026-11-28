Description: UI `links` can be relative URLs, prefixed with the base UI URL
Authors: [Doug Goldstein](https://github.com/cardoe)
Component: UI
Issues: 13479

Custom UI `links` may now be configured as relative URLs instead of requiring an absolute URL baked into each link.
When a link has no protocol, the UI prefixes it with the base UI URL.
This makes it easy to add buttons that point at other pages of the UI, for example a workflow list filtered by namespace and phase:

    links:
      - name: Failed workflows
        scope: workflow-list
        url: "?namespace=argo-events&phase=Failed&phase=Error&limit=50"
