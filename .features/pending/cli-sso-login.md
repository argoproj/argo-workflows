Description: Add `argo auth login` to log in to an SSO Argo Server from the CLI
Authors: [Ereli Eran](https://github.com/eeran-cs)
Component: CLI
Issues: 17486 14411

`argo auth login` opens your browser at the Argo Server's SSO login page and receives the token on a loopback port, encrypted to a key that only the CLI holds.
Use `argo auth login --paste` when your browser runs on another machine: you log in, copy the token from the User Info page and paste it at the prompt.
The CLI saves the token per Argo Server and uses it when `ARGO_TOKEN` isn't set, so `argo auth token` gives other tools your SSO identity.
`argo auth logout` deletes the saved token.
