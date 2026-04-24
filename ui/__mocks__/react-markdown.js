const React = require('react');
function ReactMarkdown({children}) {
    return React.createElement(React.Fragment, null, children);
}
module.exports = ReactMarkdown;
module.exports.default = ReactMarkdown;
