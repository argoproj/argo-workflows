Description: Display zoom percentage and use fixed 10% zoom steps in graph panel
Authors: [amarkdotdev](https://github.com/amarkdotdev)
Component: UI
Issues: 16257

The graph panel zoom buttons previously used a multiplicative factor which made zoom steps inconsistent and made it impossible to return to exactly 100%. This feature adds a zoom percentage badge between the zoom buttons and switches to fixed 10% increments for consistent, predictable zoom behavior.
