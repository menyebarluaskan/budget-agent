# Budget Agent — Quadlet deployment

## Prerequisites

- Linux host with cgroup v2, Podman 5.x+, systemd user services.
- `loginctl enable-linger $USER` so services run at boot.
- SELinux: use `:Z` mounts (already in the units).

## Install

```sh
mkdir -p ~/.local/share/budget-agent/data/receipts \
         ~/.local/share/budget-agent/data/backups \
         ~/.local/share/budget-agent/tailscale \
         ~/.config/budget-agent

cp deploy/quadlet/budget-agent.env.example ~/.config/budget-agent/budget-agent.env
chmod 600 ~/.config/budget-agent/budget-agent.env
# edit secrets, BASE_URL, tailnet hostname
cp deploy/quadlet/serve.json ~/.config/budget-agent/serve.json

ln -sf "$PWD/deploy/quadlet/budget-agent.pod" ~/.config/containers/systemd/
ln -sf "$PWD/deploy/quadlet/budget-agent.container" ~/.config/containers/systemd/
ln -sf "$PWD/deploy/quadlet/tailscale.container" ~/.config/containers/systemd/

systemctl --user daemon-reload
systemctl --user start budget-agent-pod.service
```

## Verify

```sh
systemctl --user status budget-agent-pod.service budget-agent.service tailscale.service
podman pod ps
journalctl --user -u budget-agent.service -f
```

## Backup

```sh
budget-agent backup --dest ~/.local/share/budget-agent/data/backups
```

Uses SQLite `VACUUM INTO`. Restore: stop the service, copy the backup file over `budget.db` (with WAL cleared), restart.

## Upgrade

1. Back up the database.
2. Build/pull the new image tag; update `Image=` in budget-agent.container.
3. `systemctl --user daemon-reload && systemctl --user restart budget-agent.service`.

## Rollback

Restore the previous image tag and the pre-upgrade backup. Destructive migrations are not allowed in a single release.

## Uninstall

```sh
systemctl --user stop budget-agent-pod.service
rm ~/.config/containers/systemd/budget-agent.{pod,container} ~/.config/containers/systemd/tailscale.container
systemctl --user daemon-reload
```
Data remains under ~/.local/share/budget-agent; remove manually if desired.
