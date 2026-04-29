Component: Build and Development
Issues: 14155
Description: Support plural "Authors:" field in feature notes
Authors: [Claude](https://github.com/anthropics)

The featuregen tool now uses the plural "Authors:" field instead of "Author:" for new feature notes.
This better reflects that features can have multiple authors.
The change maintains full backwards compatibility with existing feature files that use the singular "Author:" field.
