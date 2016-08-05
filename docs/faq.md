FAQ
===

### Resiliency concerns

> I ran `docker kill` and my container didn't restart. What gives?

Porter starts containers with `--restart=on-failure:5`. It would be pathological
for docker to ignore its own kill command. Exit the container with a non-zero
exit code (e.g. `exit 1`, throw an exception) and you'll see that containers
_are_ restarted.

> I rebooted my instance and it didn't work

Reboot isn't a real-world failure case. `cloud-init` doesn't run on an instance
reboot so none of porter's commands are invoked as they are during normal
instance initialization. Terminate your instance and you'll see that everything
works as you expect it to.
