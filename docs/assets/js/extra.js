// https://squidfunk.github.io/mkdocs-material/customization/?h=java#additional-javascript
document$.subscribe(postLoad)

function postLoad() {
  // remove version from GH link so as not to mislead readers that that version is the version of the docs they're on
  const ghVersion = document.querySelector('.md-source__fact--version'); // https://github.com/squidfunk/mkdocs-material/blob/198a6801fcf687ecb4d22e5c493fdf80427bdd33/src/templates/assets/javascripts/templates/source/index.tsx#L41
  if (ghVersion) ghVersion.remove();
}
