package buildutil

import (
	"fmt"
	"os/exec"
	"strings"
)

func doProfileCommands(cfg BuildConfig, profile string) error {
	if path := commandSurfacePath(cfg); path == "" {
		return fmt.Errorf("profile-commands requires MainPath or CommandSurfacePath")
	}

	switch profile {
	case "", "all":
		stable, err := commandSurface(cfg, nil)
		if err != nil {
			return err
		}

		dev, err := commandSurface(cfg, []string{"dev"})
		if err != nil {
			return err
		}

		printCommandProfile("stable", stable)
		fmt.Println()
		printCommandProfile("dev", dev)
		fmt.Println()

		devOnly := diffCommandSurfaces(stable, dev)
		fmt.Printf("== dev-only commands (%d commands) ==\n", len(devOnly))
		for _, path := range devOnly {
			fmt.Println(path)
		}
		return nil

	case "stable":
		commands, err := commandSurface(cfg, nil)
		if err != nil {
			return err
		}
		printCommandProfile("stable", commands)
		return nil

	case "dev":
		commands, err := commandSurface(cfg, []string{"dev"})
		if err != nil {
			return err
		}
		printCommandProfile("dev", commands)
		return nil

	default:
		return fmt.Errorf("unknown profile %q (use stable, dev, or all)", profile)
	}
}

func commandSurface(cfg BuildConfig, extraTags []string) ([]string, error) {
	args := profileCommandArgs(cfg, extraTags)

	cmd := exec.Command("go", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("collect %s command surface: %w\n%s", profileName(extraTags), err, strings.TrimSpace(string(output)))
	}

	return parseCommandSurfaceOutput(output), nil
}

func profileCommandArgs(cfg BuildConfig, extraTags []string) []string {
	args := []string{"run"}

	tags := make([]string, 0, len(cfg.BuildTags)+len(extraTags))
	tags = append(tags, cfg.BuildTags...)
	tags = append(tags, extraTags...)
	if len(tags) > 0 {
		args = append(args, "-tags", strings.Join(tags, ","))
	}

	args = append(args, commandSurfacePath(cfg))
	args = append(args, commandSurfaceArgs(cfg)...)

	return args
}

func parseCommandSurfaceOutput(output []byte) []string {
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	commands := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		commands = append(commands, line)
	}
	return commands
}

func diffCommandSurfaces(base, candidate []string) []string {
	baseSet := make(map[string]struct{}, len(base))
	for _, path := range base {
		baseSet[path] = struct{}{}
	}

	diff := make([]string, 0, len(candidate))
	for _, path := range candidate {
		if _, exists := baseSet[path]; exists {
			continue
		}
		diff = append(diff, path)
	}
	return diff
}

func printCommandProfile(name string, commands []string) {
	fmt.Printf("== %s profile (%d commands) ==\n", name, len(commands))
	for _, path := range commands {
		fmt.Println(path)
	}
}

func profileName(extraTags []string) string {
	if len(extraTags) == 0 {
		return "stable"
	}
	return strings.Join(extraTags, ",")
}
