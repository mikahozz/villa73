# Villa73 - Home Dashboard

Being rebuilt - again. Now as a monorepo including the Go backend

- frontend/ Contains the React web solution
- backend/ Contains the Go backend

## Node.js version policy

- Frontend local and Docker builds are pinned to `22.22.0`.
- `22.22.0` is used to keep Raspberry Pi `armv7` builds working.
- Node 24 images do not support `linux/arm/v7` (`armv7l`) in this setup, which causes Docker build failures on the Pi.
