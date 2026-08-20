package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	monitorapp "github.com/KageRyo/netquota/internal/app"
	"github.com/KageRyo/netquota/internal/config"
	"github.com/KageRyo/netquota/internal/format"
	"github.com/KageRyo/netquota/internal/model"
	"github.com/KageRyo/netquota/internal/network"
	"github.com/KageRyo/netquota/internal/notify"
	"github.com/KageRyo/netquota/internal/startup"
	"github.com/KageRyo/netquota/internal/storage"
	"github.com/KageRyo/netquota/internal/tray"
	"github.com/KageRyo/netquota/internal/version"
)

func main() {
	var (
		configPath     string
		statePath      string
		listInterfaces bool
		once           bool
		headless       bool
		installStartup bool
		removeStartup  bool
		showVersion    bool
	)
	flag.StringVar(&configPath, "config", "", "path to config.json")
	flag.StringVar(&statePath, "state", "", "path to state.json")
	flag.BoolVar(&listInterfaces, "list-interfaces", false, "list available network interfaces and exit")
	flag.BoolVar(&once, "once", false, "sample once, print usage, and exit")
	flag.BoolVar(&headless, "headless", false, "run monitoring without the desktop tray")
	flag.BoolVar(&installStartup, "install-startup", false, "enable start on login and exit")
	flag.BoolVar(&removeStartup, "uninstall-startup", false, "disable start on login and exit")
	flag.BoolVar(&showVersion, "version", false, "print the application version and exit")
	flag.Parse()

	if showVersion {
		fmt.Printf("NetQuota v%s\n", version.Value)
		return
	}
	if installStartup && removeStartup {
		fatal(errors.New("--install-startup and --uninstall-startup cannot be used together"))
	}

	provider := network.GopsutilProvider{}
	if listInterfaces {
		listAvailableInterfaces(provider)
		return
	}

	store := makeStore(configPath, statePath)
	cfg := loadConfig(store)
	state := loadState(store)
	executable := executablePath()
	if installStartup || removeStartup {
		enabled := installStartup
		if err := startup.Configure(enabled, executable); err != nil {
			fatal(err)
		}
		cfg.StartOnLogin = enabled
		if err := store.SaveConfig(cfg); err != nil {
			fatal(err)
		}
		if enabled {
			fmt.Println("NetQuota will start on login.")
		} else {
			fmt.Println("NetQuota will no longer start on login.")
		}
		return
	}
	if cfg.StartOnLogin {
		if err := startup.Configure(true, executable); err != nil {
			slog.Default().Warn("configure start on login", "error", err)
		}
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	monitor := monitorapp.NewMonitor(cfg, state, provider, store, store, notify.System{}, logger)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if once {
		printSample(ctx, monitor)
		return
	}
	if headless {
		runHeadless(ctx, monitor)
		return
	}
	tray.Run(ctx, monitor, executable)
}

func makeStore(configPath, statePath string) storage.Store {
	defaultConfigPath, defaultStatePath, err := config.Paths()
	if err != nil {
		fatal(err)
	}
	if configPath == "" {
		configPath = defaultConfigPath
	}
	if statePath == "" {
		statePath = defaultStatePath
	}
	return storage.Store{ConfigPath: configPath, StatePath: statePath}
}

func loadConfig(store storage.Store) model.Config {
	cfg, err := store.LoadConfig()
	if err == nil {
		return cfg
	}
	if !errors.Is(err, os.ErrNotExist) {
		fatal(err)
	}
	cfg = config.Default()
	if err := store.SaveConfig(cfg); err != nil {
		fatal(err)
	}
	return cfg
}

func loadState(store storage.Store) model.State {
	state, err := store.LoadState()
	if err == nil {
		return state
	}
	if !errors.Is(err, os.ErrNotExist) {
		fatal(err)
	}
	return model.State{Version: model.StateVersion}
}

func executablePath() string {
	path, err := os.Executable()
	if err != nil {
		fatal(fmt.Errorf("resolve executable path: %w", err))
	}
	return path
}

func listAvailableInterfaces(provider network.Provider) {
	interfaces, err := provider.Interfaces(context.Background())
	if err != nil {
		fatal(err)
	}
	for _, iface := range interfaces {
		fmt.Printf("%-20s IPv4=%-15s MAC=%s\n", iface.Name, iface.IPv4, iface.HardwareAddress)
	}
}

func printSample(ctx context.Context, monitor *monitorapp.Monitor) {
	sample, err := monitor.Sample(ctx, time.Now())
	if err != nil {
		fatal(err)
	}
	printUsage(sample)
}

func runHeadless(ctx context.Context, monitor *monitorapp.Monitor) {
	for {
		printSample(ctx, monitor)
		interval := time.Duration(monitor.Config().PollIntervalSeconds) * time.Second
		if interval < time.Second {
			interval = 2 * time.Second
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return
		case <-timer.C:
		}
	}
}

func printUsage(sample monitorapp.Sample) {
	fmt.Printf("Interface: %s\n", sample.Interface.Name)
	fmt.Printf("Download:  %s\n", format.Bytes(sample.Usage.DownloadBytes))
	fmt.Printf("Upload:    %s\n", format.Bytes(sample.Usage.UploadBytes))
	fmt.Printf("Total:     %s\n", format.Bytes(sample.Quota.Total.UsedBytes))
	if sample.Quota.Total.Enabled {
		fmt.Printf("Remaining: %s (%s)\n", format.Bytes(sample.Quota.Total.RemainingBytes), format.Percent(sample.Quota.Total.Percent))
	} else {
		fmt.Println("Remaining: total quota disabled")
	}
	if sample.DownloadReset || sample.UploadReset {
		fmt.Println("Note: an operating-system counter reset was detected; reset bytes were not counted.")
	}
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "NetQuota: %v\n", err)
	os.Exit(1)
}
