composite-action-lint
=====================


A shallow structural copy of [actionlint][actionlint-repo] 
but for linting the steps of [composite github actions][composite-action-tutorial]
instead of workflow files.


Significant portions of actionlint have been copied verbatim or with minor
changes. The actionlint license is included at
[ACTIONLINT_LICENSE.txt](./ACTIONLINT_LICENSE.txt)


## Usage

### in a workflow

Keep in mind: if you don't have any composite actions in your repo, you don't need this action.

```yaml
name: actionlint
on:
  pull_request:
    paths:
      - .github/**

jobs:
  actionlint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
      - uses: bettermarks/composite-action-lint@master
        with:
          actions: .github/actions/*/action.yml # this is the default value, so it can be omitted
```

## Installation

Installation requires a working [go][go] toolchain.

```shell
go install github.com/bettermarks/composite-action-lint/cmd/composite-action-lint@latest
```

Then ensure you have `$(go env GOPATH)/bin` on your `PATH`.

## Usage

Unlike actionlint, composite-action-lint does not search for actions to lint.
Pass each action metadata file as an argument to composite-action-lint
If the path does not point to an action file the command will fail.

```shell
composite-action-lint path/to-action/action.yml and/another/action.yaml
```

## Checks

So far only expression checks have been ported across from actionlint.

Example:

```
$ composite-action-lint testdata/examples/typo-in-input-usage/action.yml
testdata/examples/typo-in-input-usage/action.yml:11:21: property "desrciption" is not defined in object type {description: any} [expression]
   |
11 |     - run: echo ${{ inputs.desrciption }}
   |                     ^~~~~~~~~~~~~~~~~~
```

[actionlint-repo]: https://github.com/rhysd/actionlint
[composite-action-tutorial]: https://docs.github.com/en/actions/tutorials/create-actions/create-a-composite-action
[go]: https://go.dev/
