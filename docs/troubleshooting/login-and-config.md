# Login and config problems

Use this page when the CLI cannot log in or cannot find what it needs.

## `no saved Moodle session found`

Run these commands:

```sh
moodle config set \
  --username "<username>" \
  --password "<password>"
moodle login
```

Then check it:

```sh
moodle --json list courses
```

## `session expired`

Run `moodle login` again.

If the CLI does not know your username or password yet, save them first:

```sh
moodle config set --username "<username>" --password "<password>"
```

## I want to inspect the saved files

See the exact file locations in [Default paths and environment variables](../reference/paths.md).
