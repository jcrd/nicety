# nicety

[![CodeQL][codeql-badge]][codeql]

[codeql-badge]: https://github.com/jcrd/nicety/actions/workflows/codeql-analysis.yml/badge.svg
[codeql]: https://github.com/jcrd/nicety/actions/workflows/codeql-analysis.yml

nicety is a process priority management daemon for Linux that sets a process's:
  - CPU affinity
  - scheduling priority
  - I/O scheduling class and priority
  - realtime attributes

based on udev-like rules.

## Packages

* **RPM** package available from [copr][1]. [![Copr build status](https://copr.fedorainfracloud.org/coprs/jcrd/nicety/package/nicety/status_image/last_build.png)](https://copr.fedorainfracloud.org/coprs/jcrd/nicety/package/nicety/)

  Install with:
  ```
  dnf copr enable jcrd/nicety
  dnf install nicety
  ```

[1]: https://copr.fedorainfracloud.org/coprs/jcrd/nicety/

## Usage

Create rules in the `/etc/nicety/rules.d` directory.

Enable the systemd service with:

```sh
systemctl enable --now nicety
```

### Rules

Rules are JSON files with the extension `.rules`.

Example rule `/etc/nicety/rules.d/make.rules`:
```
{ "name": "make", "nice": 19, "io_class": "idle", "sched_policy": "idle" }
```

Valid keys:
- `name`: the name of the process command as given in `/proc/<PID>/comm`
  (required)
- `cpu_affinity`: bond a process to a given set of CPUs ([man page][taskset])
- `nice`: alter the scheduling priority ([man page][renice])
- `io_class`: set I/O scheduling class ([man page][ionice])
- `io_priority`: set I/O scheduling priority ([man page][ionice])
- `sched_policy`: set realtime scheduling policy ([man page][chrt])
- `sched_priority`: set realtime scheduling priority ([man page][chrt])
- `delay`: delay after which the above attributes are applied if the process
  is still running

[taskset]: https://www.commandlinux.com/man-page/man1/taskset.1.html
[renice]: https://www.commandlinux.com/man-page/man1/renice.1.html
[ionice]: https://www.commandlinux.com/man-page/man1/ionice.1.html
[chrt]: https://www.commandlinux.com/man-page/man1/chrt.1.html

## License

This project is licensed under the MIT License (see [LICENSE](LICENSE)).
