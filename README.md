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

The Windows service layer implements a pseudo-Windows service that mimics the behavior of a real service, but does not actually run as a service.

- This is required because tray icons are not supported by Windows services, as they run in the system space, and cannot interact with the user space.

Currently, I only have a MSI installer developed for Windows. I'm considering creating a specialized CLI-based installation method for Windows, one that will match the Linux experience, but that is yet to be completed.

### Linux Service Layer

The Linux service layer implements a systemd service 'notify' type service.

- Note that we don't take advantage of most modern systemd features, such as `notify-reload`, `ReloadSignal=SIGHUP`, and so on.
  - This is because I use WSL2 as my primary development environment, which only has systemd v249.
- It uses the go-systemd package to interface with systemd, enabling proper handling of startup and reload signals.
- The unit file is configured to send `SIGHUP` signals on reload, and will respond to `SIGHUP` (reload), and `SIGTERM` (stop). It also provides on-startup status updates, a watchdog mechanism, and a heartbeat mechanism (that updates the service status regularly).

Currently, the Linux service layer is only installed via the `task service` command.

Ideally, I plan to provide at least two different methods for installation:

- A one-command remote bash script that will download the binary, install the systemd unit file, and start the service.
- An internal CLI-based method that provides customized systemd unit file generation & simple management commands.

### Feature Targets

- [x] Cross-platform Background Service (Linux, Windows)
- [ ] Windows
  - [x] MSI-based Installer
  - [ ] CLI-based Installer
  - [ ] Winget Package Publishing
- [ ] Linux
  - [x] `systemd` Service Implementation
  - [ ] CLI-based Installer
  - [ ] Script-based Installer
  - [ ] `systemd` Unit File Templating/Generation
  - [ ] Smart `journalctl` logging bypass
- Application
  - [ ] TOML Configuration
  - [x] Health Checks
  - [x] Tray Icon
  - [ ] Tray Menu
  - [x] Structured Logging
    - [ ] Configurable
    - [ ] Better library (logrus, zap, zerolog, etc.)
  - [ ] Testing
    - [ ] Unit Tests
    - [ ] Integration Tests
    - [ ] Code Coverage
- [x] Development Tooling
- [x] Conventional Commits
- [x] GitHub Actions
  - [x] Per-commit Artifacts
  - [x] MSI Packages
  - [ ] Automatic Releases (GitHub Releases, Winget)
  - [ ] Testing, Linting, and/or Formatting
- [ ] README Documentation Links
