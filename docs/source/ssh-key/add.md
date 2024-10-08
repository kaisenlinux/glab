---
stage: Create
group: Code Review
info: To determine the technical writer assigned to the Stage/Group associated with this page, see https://about.gitlab.com/handbook/product/ux/technical-writing/#assignments
---

<!--
This documentation is auto generated by a script.
Please do not edit this file directly. Run `make gen-docs` instead.
-->

# `glab ssh-key add`

Add an SSH key to your GitLab account.

## Synopsis

Creates a new SSH key owned by the currently authenticated user.

Requires the '--title' flag.

```plaintext
glab ssh-key add [key-file] [flags]
```

## Examples

```plaintext
# Read ssh key from stdin and upload.
$ glab ssh-key add -t "my title"

# Read ssh key from specified key file and upload.
$ glab ssh-key add ~/.ssh/id_ed25519.pub -t "my title"

```

## Options

```plaintext
  -e, --expires-at string   The expiration date of the SSH key. Uses ISO 8601 format: YYYY-MM-DDTHH:MM:SSZ.
  -t, --title string        New SSH key's title.
```

## Options inherited from parent commands

```plaintext
      --help              Show help for this command.
  -R, --repo OWNER/REPO   Select another repository. Can use either OWNER/REPO or `GROUP/NAMESPACE/REPO` format. Also accepts full URL or Git URL.
```
