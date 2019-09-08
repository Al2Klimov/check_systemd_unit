## About

The check plugin **check\_systemd\_unit** monitors a systemd unit.

## Usage

The [plug-and-play Linux binaries]
take the following CLI arguments and no environment variables:

```
$ ./check_systemd_unit -unit UNIT \
[-warn PROPERTY(THRESHOLD) ...] [-crit PROPERTY(THRESHOLD) ...] \
[-js-warn JS_EXPR ...] [-js-crit JS_EXPR ...]
```

UNIT is the systemd unit to monitor, e.g. `icinga2.service`.

PROPERTY(THRESHOLD) specifies a numeric unit property and
an alert threshold range conforming to the [Nagio$ check plugin API],
e.g. `-warn NRestarts(@~:42)` warns if the unit's NRestarts are <= 42.

JS_EXPR is an alert condition written in [JavaScript],
e.g. `-js-crit ActiveState==="active"` returns status critical
if the unit's ActiveState is **not** "active".

### Legal info

To print the legal info, execute the plugin in a terminal:

```
$ ./check_systemd_unit
```

In this case the program will always terminate with exit status 3 ("unknown")
without actually checking anything.

### Testing

If you want to actually execute a check inside a terminal,
you have to connect the standard output of the plugin to anything
other than a terminal – e.g. the standard input of another process:

```
$ ./check_systemd_unit -unit icinga2.service |cat
```

In this case the exit code is likely to be the cat's one.
This can be worked around like this:

```
bash $ set -o pipefail
bash $ ./check_systemd_unit -unit icinga2.service |cat
```

### Actual monitoring

Just integrate the plugin into the monitoring tool of your choice
like any other check plugin. (Consult that tool's manual on how to do that.)
It should work with any monitoring tool
supporting the [Nagio$ check plugin API].

The only limitation: check\_systemd\_unit must be run on the host
to be checked – either with an agent of your monitoring tool or by SSH.
Otherwise it will check the host your monitoring tool runs on.

#### Icinga 2

This repository ships the [check command definition]
as well as a [service template] and [host example] for [Icinga 2].

The service definition will work in both correctly set up [Icinga 2 clusters]
and Icinga 2 instances not being part of any cluster
as long as the [hosts] are named after the [endpoints].

[plug-and-play Linux binaries]: https://github.com/Al2Klimov/check_systemd_unit/releases
[Nagio$ check plugin API]: https://nagios-plugins.org/doc/guidelines.html#AEN78
[JavaScript]: https://developer.mozilla.org/en-US/docs/Web/JavaScript
[check command definition]: ./icinga2/check_systemd_unit.conf
[service template]: ./icinga2/check_systemd_unit-service.conf
[host example]: ./icinga2/check_systemd_unit-host.conf
[Icinga 2]: https://www.icinga.com/docs/icinga2/latest/doc/01-about/
[Icinga 2 clusters]: https://www.icinga.com/docs/icinga2/latest/doc/06-distributed-monitoring/
[hosts]: https://www.icinga.com/docs/icinga2/latest/doc/09-object-types/#host
[endpoints]: https://www.icinga.com/docs/icinga2/latest/doc/09-object-types/#endpoint
