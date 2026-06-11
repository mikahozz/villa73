# How to move from current SD only setup into using HDD only.

bootboot Current state:

┌────────────────┬──────────┬─────────────┬────────────────┬──────────────────────────┐
│ Device │ Size │ Label │ Mounted │ Purpose │
├────────────────┼──────────┼─────────────┼────────────────┼──────────────────────────┤
│ mmcblk0p1 (SD) │ 512 MB │ system-boot │ /boot/firmware │ Boot partition (active) │
├────────────────┼──────────┼─────────────┼────────────────┼──────────────────────────┤
│ mmcblk0p2 (SD) │ 59.2 GB │ writable │ / │ Root filesystem (active) │
├────────────────┼──────────┼─────────────┼────────────────┼──────────────────────────┤
│ sda1 (HDD) │ 256 MB │ boot │ — │ Not mounted │
├────────────────┼──────────┼─────────────┼────────────────┼──────────────────────────┤
│ sda2 (HDD) │ 223.3 GB │ rootfs │ — │ Not mounted │
└────────────────┴──────────┴─────────────┴────────────────┴──────────────────────────┘

Key observations:

- The OS, Docker (/var/lib/docker), and everything else currently runs entirely from the SD card.
- The HDD (/dev/sda, 223.6 GB) has two partitions that appear to have been imaged previously (labels "boot"/"rootfs"), but is completely unused — not mounted anywhere.
- The Pi boots via root=LABEL=writable in cmdline.txt, meaning it will always find and use the SD card root unless changed.
- vcgencmd and rpi-eeprom-config aren't available, so EEPROM boot order can't be checked from this shell — you'd need to run those interactively with sudo.

---

Plan to move everything to the HDD:

1. Format the HDD

2. Clone SD → HDD

Use rsync to copy the live root filesystem from mmcblk0p2 → sda2, and copy the boot partition mmcblk0p1 → sda1. This preserves all your current config, Docker setup, and this project.

3. Make the HDD bootable

After cloning, update /boot/cmdline.txt on sda1 so it says root=LABEL=rootfs (pointing to sda2, not the SD card's "writable" partition). Also update /etc/fstab on sda2 to reference sda2 by UUID/label.

4. Test boot from HDD

Change EEPROM boot order to prefer USB/SATA

Run sudo rpi-eeprom-config --edit (needs rpi-eeprom package installed) and set BOOT_ORDER=0xf14 (try SD first → USB/SATA → repeat). Or set 0xf41 to try HDD first, SD as fallback. This requires interactive sudo in a terminal.

Reboot. The Pi should pick up the HDD. Verify with lsblk that / is on sda2.

5. Test docker from homeapp73-docker folder with: `docker compose up -d`

# Fine-tuned plan

What I found:

- Current root: mmcblk0p2 (label=writable), boot: mmcblk0p1 (label=system-boot)
- HDD already has: sda1 (label=boot), sda2 (label=rootfs) — these labels are perfect
- rpi-eeprom is installed at /usr/bin/rpi-eeprom-config
- Docker has no data yet (4KB), so nothing special needed there
- SD card uses 16G of 59G, HDD sda2 is 223G — plenty of space

Adjusted execution plan:

1. ~~Format HDD~~ — already done
2. Mount sda1 + sda2, rsync root (/ → sda2) and boot (/boot/firmware/ → sda1)
3. Fix cmdline.txt on sda1: root=LABEL=writable → root=LABEL=rootfs
4. Fix /etc/fstab on sda2: update both labels (writable→rootfs, system-boot→boot)
5. Set EEPROM boot order to USB-first via rpi-eeprom-config
6. Reboot — verify / lands on sda2
