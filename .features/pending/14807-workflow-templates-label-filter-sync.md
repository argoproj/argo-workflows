Component: UI
Issues: 14807
Description: Add label query parameter sync with URL in WorkflowTemplates UI to match Workflows list behavior for consistent filtering.
Author: [puretension](https://github.com/puretension)

WorkflowTemplates UI now properly handles label query parameters (e.g., ?label=key%3Dvalue)
Combined URL updates and localStorage persistence in single useEffect
Enables custom UI links for filtered template views
Verified that URL updates when changing filters and filters persist on page refresh
