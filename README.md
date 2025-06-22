# HASS Tray - Windows Service

A minimal Go application that can run as a Windows service with JSON logging.

## Features

- Runs as a Windows service
- JSON logging with debug level
- Logs to `current.log` in the same directory as the executable
- Heartbeat logging every 5 seconds
- Interactive mode for testing

## Requirements

- Go 1.21 or later
- Windows operating system
- Administrator privileges (for service installation)
- Task (taskfile.dev) - modern task runner

## Installing Task

Download and install Task from [taskfile.dev](https://taskfile.dev/installation/):

```bash
# Using Chocolatey
choco install task

# Using Scoop
scoop install task

# Using winget
winget install Task.Task

# Or download directly from GitHub releases
```

## Building

```bash
task build
```

This will create `hass-tray.exe` in the current directory.

## Installation

```bash
task install
```

This will:
1. Build the application
2. Copy the executable to `%APPDATA%\hass-tray\`
3. Display instructions for installing as a Windows service

To install as a Windows service (run as Administrator):

```cmd
task service-install
```

Or manually:
```cmd
sc create hass-tray binPath= "%APPDATA%\hass-tray\hass-tray.exe" start= auto
sc description hass-tray "Home Assistant Tray Service"
sc start hass-tray
```

## Uninstallation

```bash
task uninstall
```

This will:
1. Stop and remove the Windows service
2. Delete the executable from the AppData directory
3. Remove the installation directory

## Testing

To run the application in interactive mode for testing:

```bash
task run
```

This will run the application directly and show debug output in the console.

## Logging

The application logs JSON-formatted messages to `current.log` in the same directory as the executable. Log entries include:

- Timestamp in RFC3339 format
- Log level (debug)
- Message content
- Service name

Example log entry:
```json
{"timestamp":"2024-01-15T10:30:00Z","level":"debug","message":"Service heartbeat","service":"hass-tray"}
```

## Service Management

Once installed as a service, you can manage it using Task commands:

- Install service: `task service-install` (requires admin)
- Uninstall service: `task service-uninstall` (requires admin)
- Start service: `task service-start`
- Stop service: `task service-stop`
- Check status: `task service-status`

Or using standard Windows commands:

- Start: `sc start hass-tray`
- Stop: `sc stop hass-tray`
- Status: `sc query hass-tray`
- Delete: `sc delete hass-tray`

## Available Task Commands

- `task build` - Build the application
- `task install` - Build and copy to AppData directory
- `task uninstall` - Remove service and files
- `task clean` - Remove build artifacts
- `task run` - Build and run in interactive mode
- `task test` - Run tests
- `task fmt` - Format Go code
- `task vet` - Vet Go code
- `task deps` - Download and tidy dependencies
- `task service-install` - Install as Windows service (requires admin)
- `task service-uninstall` - Uninstall Windows service (requires admin)
- `task service-start` - Start the service
- `task service-stop` - Stop the service
- `task service-status` - Check service status
- `task dev` - Complete development workflow
- `task` - Show all available tasks

## Development

The application uses the `golang.org/x/sys/windows/svc` package for Windows service functionality. The main service logic is in the `Execute` method of the `myservice` struct.

When running in interactive mode (not as a service), the application will run the same logic but in the foreground with console output.

## Why Task instead of Make?

Task provides several advantages over Make:

- **Better Windows support** - Native Windows compatibility
- **YAML configuration** - More readable and maintainable
- **Cross-platform** - Works on Windows, macOS, and Linux
- **Dependencies** - Better task dependency management
- **Variables** - More flexible variable handling
- **Parallel execution** - Built-in support for parallel task execution
- **Silent mode** - Better control over output verbosity 