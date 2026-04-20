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

	profiles, err := commandSurfaceProfiles(cfg)
	if err != nil {
		return err
	}

	switch profile {
	case "", "all":
		surfaces := make(map[string][]string, len(profiles))
		for i, commandProfile := range profiles {
			commands, err := commandSurface(cfg, commandProfile)
			if err != nil {
				return err
			}
			surfaces[commandProfile.Name] = commands
			printCommandProfile(commandProfile.Name, commands)
			if i < len(profiles)-1 {
				fmt.Println()
			}
		}

		baseline := profiles[0]
		for _, commandProfile := range profiles[1:] {
			fmt.Println()
			diff := diffCommandSurfaces(surfaces[baseline.Name], surfaces[commandProfile.Name])
			fmt.Printf("== %s-only commands vs %s (%d commands) ==\n", commandProfile.Name, baseline.Name, len(diff))
			for _, path := range diff {
				fmt.Println(path)
			}
		}

		return nil

	default:
		commandProfile, ok := findCommandSurfaceProfile(profiles, profile)
		if !ok {
			return fmt.Errorf("unknown profile %q (use all or one of: %s)", profile, strings.Join(commandSurfaceProfileNames(profiles), ", "))
		}

		commands, err := commandSurface(cfg, commandProfile)
		if err != nil {
			return err
		}
		printCommandProfile(commandProfile.Name, commands)
		return nil
	}
}

func commandSurface(cfg BuildConfig, profile CommandSurfaceProfile) ([]string, error) {
	args := profileCommandArgs(cfg, profile)

	cmd := exec.Command("go", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("collect %s command surface: %w\n%s", profile.Name, err, strings.TrimSpace(string(output)))
	}

	return parseCommandSurfaceOutput(output), nil
}

func profileCommandArgs(cfg BuildConfig, profile CommandSurfaceProfile) []string {
	args := []string{"run"}

	profileCfg := cfg
	profileCfg.BuildTags = append(append([]string{}, cfg.BuildTags...), profile.Tags...)
	args = append(args, buildTagsArgs(profileCfg)...)

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

func findCommandSurfaceProfile(profiles []CommandSurfaceProfile, name string) (CommandSurfaceProfile, bool) {
	for _, profile := range profiles {
		if profile.Name == name {
			return profile, true
		}
	}
	return CommandSurfaceProfile{}, false
}

func commandSurfaceProfileNames(profiles []CommandSurfaceProfile) []string {
	names := make([]string, 0, len(profiles))
	for _, profile := range profiles {
		names = append(names, profile.Name)
	}
	return names
}

func printCommandProfile(name string, commands []string) {
	fmt.Printf("== %s profile (%d commands) ==\n", name, len(commands))
	for _, path := range commands {
		fmt.Println(path)
	}
}
