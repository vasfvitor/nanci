# Build Directory

Build configuration and assets of Nanci Desktop for Wails v3 (v3.0.0). The app ships for Windows only.

* `config.yml` - project information (company, product, version, copyright) and the `wails3 dev` settings.
* `Taskfile.yml` - common tasks: frontend install and build, bindings generation, icons, `update:build-assets`.
* `windows/` - Windows files and tasks.
* `darwin/` - macOS plists, kept as `wails3 update build-assets` writes them; there is no macOS build.
* `appicon.png` - source image for the icons.

The build output goes to `../bin` (`nanci-desktop.exe` and the installer).

## Generated from config.yml

`windows/info.json`, `windows/wails.exe.manifest`, `windows/nsis/wails_tools.nsh` and the `darwin` plists are generated from `config.yml`:

```bash
wails3 task common:update:build-assets
```

Run it after changing `info` in `config.yml`, then delete the `ios/` and `linux/` folders the command also writes. The release workflow runs it after writing the tag version to `config.yml`.

## Windows

- `icon.ico` - the application icon. It is generated from `appicon.png` only when missing.
- `info.json` - the version information of the executable (right click the exe -> Properties -> Details).
- `wails.exe.manifest` - the application manifest (per-monitor DPI awareness).
- `nsis/project.nsi` - the installer script, customised for Nanci: zlib compression, the `NanciDesktop` uninstall key, a desktop shortcut and no file or protocol associations. `windows:package` installs per user (`INSTALL_SCOPE=user`, no UAC prompt).
- `nsis/wails_tools.nsh` - generated helpers for the installer (do not edit).
- `Taskfile.yml` - `windows:build` (the production `-ldflags` take `EXTRA_LDFLAGS`) and `windows:package`.
