Description: Add copy button to workflow logs viewer
Authors: [nakatani-yo](https://github.com/nakatani-yo)
Component: UI
Issues: 636, 6384

The workflow logs viewer now includes a "Copy" button in the toolbar that copies the entire log content to the clipboard.
The button provides visual feedback by briefly showing a checkmark icon and "Copied" text after a successful copy.
This resolves a long-standing UX pain point where manually selecting log text was error-prone, especially for running pods where the selection gets cleared as new logs stream in.
