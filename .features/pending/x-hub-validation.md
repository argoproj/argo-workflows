Component: General
Issues: 15064
Description: Websub X-Hub-Signature header validation
Author: [Alessandro Ogier](https://github.com/aogier)

To easy implement webhook with more platform which can sign their
payloads, [WebSub
standard](https://www.w3.org/TR/websub/#authenticated-content-distribution)
is now implemented, plus a non-standard feature for base64 encoded
binary signatures.
