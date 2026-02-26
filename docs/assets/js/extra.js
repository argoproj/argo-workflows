// https://squidfunk.github.io/mkdocs-material/customization/?h=java#additional-javascript
document$.subscribe(postLoad)

function postLoad() {
  removeGHVersion();
  addVersionToLogo();
}

// remove version from GH link so as not to mislead readers that that version is the version of the docs they're on
function removeGHVersion() {
  // Selector: https://github.com/squidfunk/mkdocs-material/blob/198a6801fcf687ecb4d22e5c493fdf80427bdd33/src/templates/assets/javascripts/templates/source/index.tsx#L41
  // Used in Header (https://github.com/squidfunk/mkdocs-material/blob/198a6801fcf687ecb4d22e5c493fdf80427bdd33/src/templates/partials/header.html#L106) and SideNav (https://github.com/squidfunk/mkdocs-material/blob/198a6801fcf687ecb4d22e5c493fdf80427bdd33/src/templates/partials/nav.html#L58)
  document.querySelectorAll('.md-source__fact--version').forEach(removeGHVersionElem);
}

function removeGHVersionElem(ghVersion) {
  ghVersion.remove();
}

function addVersionToLogo() {
  // Selectors: Header (https://github.com/squidfunk/mkdocs-material/blob/198a6801fcf687ecb4d22e5c493fdf80427bdd33/src/templates/partials/header.html#L42) and SideNav (https://github.com/squidfunk/mkdocs-material/blob/198a6801fcf687ecb4d22e5c493fdf80427bdd33/src/templates/partials/nav.html#L46)
  document.querySelectorAll('.md-logo').forEach(addVersionToLogoElem);
}

// add version underneath logo
function addVersionToLogoElem(logo, index) {
  const isSideNav = index == 1; // 0 is header, 1 is sidenav

  // append a span with text content containing the minor version
  const versionSpan = document.createElement('span');
  versionSpan.appendChild(document.createTextNode(getMinorVersionFromUrl()));
  versionSpan.style.fontSize = '12px'; // small but not tiny
  if (isSideNav) {
    versionSpan.style.position = 'absolute';
    versionSpan.style.top = '20%'; // align with the argonaut logo's "ears" (top of text starts at 20%)
    versionSpan.style.left = '65px'; // logo is 48 x 48 plus a margin; rough eyeballed calculation
  }
  logo.appendChild(versionSpan);

  // modify the parent's style a bit to account for the new text
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
