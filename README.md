# HATray

A simple tray utility for Home Assistant

- Display door, window, motion or other sensor states as icon within your tray.
- Instantaneous updates via WebSocket connections
- Configurable via TOML
- Easy to install, runs as a background service
- Cross-platform support (Windows, Linux)

## Design

The application follows a layered architecture:

- **Command Layer**: A barebones entrypoint for the application, initializing the logger & emitting basic diagnostics.
- **Service Layer**: OS-dependent implementation that communicates with the App layer. It contains the true entrypoint for the application.
- **App Layer**: Generic, cross-platform implementation that exposes simple methods for controlling the application state
    - **Pause**: Disconnect from the server and cease any background tasks.
        - Once paused, no logging occurs from the App layer, no connections are made, and no background tasks should run.
    - **Resume**: Reads configuration files, connects to the server and initiates background tasks.
        - Once running, the App layer should be connected (or attempting to reconnect) to the server.
        - If an error occurs while attempting to resume or while running, the app layer will become paused.
    - **Reload**: If not paused, pause the application, re-read configuration files, then resume. This is just a macro for pause + resume.

### Windows Service Layer

The Windows service layer implements a legitimate Windows service that receives control signals through the Windows Service Control Manager (SCM). It uses Go project's [/x/sys/](https://pkg.go.dev/golang.org/x/sys) module (for [/x/sys/windows/svc](https://pkg.go.dev/golang.org/x/sys/windows/svc)) to interface with the SCM.

- In development (i.e. ran directly, or with `go run`), the service layer detects this and runs as a 'debug' service, which imitates the behavior of a real service, but does not actually run as a service.
- The service is fully responsive to most standard commands, including Start, Stop, Pause, Continue, and Interrogate.

For local development, you can run and build directly, or use `task service` to quickly install a service. `task package` provides a quick way to build and package the application using WiX.

Currently, I only have a MSI installer developed for Windows. I'm considering creating a specialized CLI-based installation method for Windows, one that will match the Linux experience, but that is yet to be completed.

### Linux Service Layer

The Linux service layer implements a systemd service 'notify' type service.
- Note that we don't take advantage of most modern systemd features, such as `notify-reload`, `ReloadSignal=SIGHUP`, and so on.
    - This is because I use WSL2 as my primary development environment, which only has systemd v249.
- It uses the go-systemd package to interface with systemd, enabling proper handling of startup and reload signals.
- The unit file is configured to send `SIGHUP` signals on reload, and will respond to `SIGHUP` (reload),  and `SIGTERM` (stop). It also provides on-startup status updates, a watchdog mechanism, and a heartbeat mechanism (that updates the service status regularly).

Currently, the Linux service layer is only installed via the `task service` command.

Ideally, I plan to provide at least two different methods for installation:
- A one-command remote bash script that will download the binary, install the systemd unit file, and start the service.
- An internal CLI-based method that provides customized systemd unit file generation & simple management commands.

### Feature Targets

- [X] Cross-platform Background Service (Linux, Windows)
- [ ] Windows
    - [X] Windows Service Implementation
    - [X] MSI-based Installer
    - [ ] CLI-based Installer
    - [ ] Winget Package Publishing
- [ ] Linux
    - [X] `systemd` Service Implementation
    - [ ] CLI-based Installer
    - [ ] Script-based Installer
    - [ ] `systemd` Unit File Templating/Generation
    - [ ] Smart `journalctl` logging bypass
- Application
    - [ ] TOML Configuration
    - [ ] Health Checks
    - [ ] Tray Icon, Tray Menu
    - [X] Structured Logging
        - [ ] Configurable
        - [ ] Better library (logrus, zap, zerolog, etc.)
    - [ ] Testing
        - [ ] Unit Tests
        - [ ] Integration Tests
        - [ ] Code Coverage
- [X] Development Tooling
- [X] Conventional Commits
- [X] GitHub Actions
    - [X] Per-commit Artifacts
    - [X] MSI Packages
    - [ ] Automatic Releases (GitHub Releases, Winget)
    - [ ] Testing, Linting, and/or Formatting
- [ ] README Documentation Links