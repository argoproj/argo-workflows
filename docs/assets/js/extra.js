// https://squidfunk.github.io/mkdocs-material/customization/?h=java#additional-javascript
document$.subscribe(postLoad)

function postLoad() {
  removeGHVersion();
  addVersionToLogo();
}

// remove version from GH link so as not to mislead readers that that version is the version of the docs they're on
function removeGHVersion() {
  const ghVersion = document.querySelector('.md-source__fact--version'); // https://github.com/squidfunk/mkdocs-material/blob/198a6801fcf687ecb4d22e5c493fdf80427bdd33/src/templates/assets/javascripts/templates/source/index.tsx#L41
  if (ghVersion) ghVersion.remove();
}

// add version underneath logo
function addVersionToLogo() {
  const logo = document.querySelector('.md-logo'); // https://github.com/squidfunk/mkdocs-material/blob/198a6801fcf687ecb4d22e5c493fdf80427bdd33/src/templates/partials/header.html#L42
  if (!logo) return;

  // append a span with text content containing the minor version
  const versionSpan = document.createElement('span');
  versionSpan.appendChild(document.createTextNode(getMinorVersionFromUrl()));
  logo.appendChild(versionSpan);
  // modify the style a bit to account for the new text
  logo.style.textAlign = 'center';
  logo.style.marginBottom = '.1rem'; // shrink bottom margin a bit from original .2rem (https://github.com/squidfunk/mkdocs-material/blob/198a6801fcf687ecb4d22e5c493fdf80427bdd33/src/templates/assets/stylesheets/main/components/_header.scss#L104)
}


// pull the minor version out of the URL, assuming it has `/release-3.5/` or similar in it. default to 'dev'
function getMinorVersionFromUrl() {
  const currentUrl = window.location.href;
  const releaseIndex = currentUrl.indexOf('release-');
  if (releaseIndex == -1) return 'dev';

  const afterRelease = currentUrl.split(releaseIndex)[1];
  return afterRelease.split('/')?.[0] ?? 'dev'
}
