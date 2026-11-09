# Runtime-only deployment

Use `.deployignore` from the matching source root as an rsync merge filter.
These files are rsync filters, not Git ignore files. Only runtime files are
included; local source, tests, Node dependencies, screenshots, temporary
binaries, setup/migration helpers, and private credentials are excluded.

Web preview (run from the repository root):

```sh
rsync -avni --filter='merge web/.deployignore' web/ root@76.13.195.48:/www/wwwroot/pastrack/
```

Receiver preview (requires building the Linux binary as `pastrack` first):

```sh
rsync -avni --filter='merge .deployignore' ./ root@76.13.195.48:/opt/pastrack/
```

These are previews only. For binary updates, transfer to a staging directory,
then stop the matching aaPanel project, atomically replace its executable,
and start/check that project as described in the project README. Never rsync
directly over a running executable. Do not restart the receiver for web-only
updates. No backend code is changed by these filters.

Never add `--delete-excluded`: server-only `.env`, spool, runtime sockets and
credentials must survive. The filters do not remove files already on the
server. Audit those separately against launch commands, environment file
references, open files and the running binary before deleting explicit paths.
Preserve `/etc/pastrack`, logs, backups and legacy rollback files unless
separately reviewed. Firebase credentials may be referenced by
`GOOGLE_APPLICATION_CREDENTIALS`; an excluded JSON file is not evidence that
its server copy is unused. Keep one-time migration helpers locally and stage
them privately only when needed, removing the staged copy afterward.
