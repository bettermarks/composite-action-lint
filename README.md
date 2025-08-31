composite-action-lint
=====================


A shallow structural copy of [actionlint][actionlint-repo] but instead for
linting the steps of [composite github actions][composite-action-tutorial]
instead of github workflow files.


Significant portions of actionlint have been copied verbatim or with minor
changes. The actionlint license is included at
[ACTIONLINT_LICENSE.txt](./ACTIONLINT_LICENSE.txt)


## Installation

Installation requires a working [go][go] toolchain.

```shell
go install github.com/bettermarks/composite-action-lint/cmd/composite-action-lint@latest
```

## Usage

Unlike actionlint, composite-action-lint does not search for actions to lint.
Pass each action metadata file as an argument to composite-action-lint

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
