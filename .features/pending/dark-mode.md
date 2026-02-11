Description: Add dark mode support to the UI
Authors: [Clint Moyer](https://github.com/clintmoyer)
Component: UI
Issues: 5037

Dark mode allows users to switch between light, dark, and system-preference themes.
The theme selector is available on the User Info page with three options: Light, Dark, and System.

The theme preference is persisted to localStorage and automatically applied on page load.
When "System" is selected, the UI follows the operating system's color scheme preference.

This feature matches the dark mode implementation in Argo CD.
