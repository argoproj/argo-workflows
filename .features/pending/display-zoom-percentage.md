Description: Display zoom percentage and use fixed 10% zoom steps in graph panel
Authors: [nakatani-yo](https://github.com/nakatani-yo)
Component: UI
Issues: 16257

The graph panel now displays the current zoom percentage as a badge next to the zoom buttons.
Zoom in/out uses fixed 10% steps of the base node size instead of a multiplicative factor, ensuring consistent behavior and allowing users to return to exactly 100%.
A minimum zoom limit of 10% prevents nodes from becoming invisible.
