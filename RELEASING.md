# Releasing the TofuGG CasaOS fork

The installer (`install.sh`) and updater (`update`) download per-component
release tarballs from the **TofuGG** GitHub org with the asset names below.
Publishing requires **one GitHub release per component** with a **tag equal to
the version** and **one tarball attached** — exactly how the old
`get.casaos.io` pipeline worked.

## Component → release mapping (current batch: v0.4.18/v0.4.7)

| Repo | Tag | Asset name |
|------|-----|------------|
| TofuGG/CasaOS | `v0.4.18` | `linux-amd64-casaos-v0.4.18.tar.gz` |
| TofuGG/CasaOS-UI | `v0.4.7` | `linux-all-casaos-v0.4.7.tar.gz` |
| TofuGG/CasaOS-Gateway | `v0.4.10` | `linux-amd64-casaos-gateway-v0.4.10.tar.gz` |
| TofuGG/CasaOS-UserService | `v0.4.10` | `linux-amd64-casaos-user-service-v0.4.10.tar.gz` |
| TofuGG/CasaOS-AppManagement | `v0.4.18` | `linux-amd64-casaos-app-management-v0.4.18.tar.gz` |

Unchanged upstream components (MessageBus, LocalStorage, CLI, AppStore) continue
to be fetched from the IceWhaleTech releases by the scripts — no fork release
needed for those.

> If a fork component ever ships with a `-fork.N` suffix tag (e.g.
> `v0.4.18-fork.1`), the version-check in `service/casa.go` normalizes it to
> `0.4.18`, so the in-UI "update available" badge compares base versions only.
> The asset name above must still match the script's pinned tag string exactly.

## Tarball layout (required by the scripts)

The scripts `curl` each asset and `tar zxf`, then use `BUILD_DIR=…/build`,
`$BUILD_DIR/sysroot`, `$BUILD_DIR/scripts/migration/script.d` and
`$BUILD_DIR/scripts/setup/script.d`. So every tarball is compressed **from the
repo root and contains a single top-level `build/` directory**:

```sh
# in the repo root (Linux preferred so exec bits survive):
tar czf linux-amd64-casaos-v0.4.18.tar.gz build
```

Each Go component's tarball must include the compiled binary at
`build/sysroot/usr/bin/<binary>` (`casaos`, `casaos-gateway`,
`casaos-user-service`, `casaos-app-management`) — the repo `build/` checkout
does **not** contain the binary; add it before packing:

```sh
mkdir -p build/sysroot/usr/bin
cp casaos build/sysroot/usr/bin/casaos
chmod +x build/sysroot/usr/bin/casaos
tar czf linux-amd64-casaos-v0.4.18.tar.gz build
```

The UI tarball is arch-less (`linux-all-…`) and packages the compiled dashboard
under `build/sysroot/var/lib/casaos/www` (the `npm run build` output) plus its
`build/scripts` so the updater can run UI migration/setup hooks.

## Publready artifacts from this repo's build

Pre-built, publish-ready tarballs (Linux tar, exec bits set) are produced by:

```sh
# from a checked-out repo with binaries built:
docker run --rm -v "$PWD:/publish" debian:bookworm-slim bash /publish/make-tarballs.sh
```

and land in the local output directory. Attach each one to its component's
release. Release the **CasaOS** repo *last* among the three backend releases so
a version-check bump matches what the installer can actually download.

## Before publishing

1. Bump each component's in-code version to match the tag
   (`common/constants.go` for CasaOS/AppManagement, `common/version.go` for
   Gateway/UserService, `package.json` for UI).
2. Commit + push the code with a matching tag.
3. Create the release with the asset; mark **pre-release** if it is a soak
   build (the version-check handles pre-releases).
4. Never point update/install at `get.casaos.io` — that pipeline is upstream
   and overwrites fork features.