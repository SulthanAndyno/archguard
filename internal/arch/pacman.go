package arch

import (
	"bufio"
	"bytes"
	"context"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// PackageUpdate represents a package available for upgrade.
type PackageUpdate struct {
	Name       string
	OldVersion string
	NewVersion string
	IsCritical bool
}

// OrphanPackage represents an unneeded dependency package.
type OrphanPackage struct {
	Name        string
	Description string
	Size        string
	Selected    bool
}

// CriticalPackages is a set of core packages that warrant special attention when upgrading.
var CriticalPackages = map[string]bool{
	"linux":          true,
	"linux-zen":      true,
	"linux-lts":      true,
	"linux-hardened": true,
	"systemd":        true,
	"systemd-libs":   true,
	"glibc":          true,
	"nvidia":         true,
	"nvidia-utils":   true,
	"mesa":           true,
	"btrfs-progs":    true,
	"pipewire":       true,
	"wireplumber":    true,
	"networkmanager": true,
	"openssl":        true,
	"sudo":           true,
	"pacman":         true,
	"mkinitcpio":     true,
}

// GetInstalledPackages returns a fast lookup set of all explicitly and implicitly installed packages.
func GetInstalledPackages(ctx context.Context) (map[string]bool, error) {
	cmd := exec.CommandContext(ctx, "pacman", "-Qq")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	installed := make(map[string]bool)
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		pkg := strings.TrimSpace(scanner.Text())
		if pkg != "" {
			installed[pkg] = true
		}
	}
	return installed, nil
}

// GetPendingUpdates executes checkupdates to list packages ready for upgrade.
func GetPendingUpdates(ctx context.Context) ([]PackageUpdate, error) {
	// checkupdates comes standard from pacman-contrib and avoids lock issues
	cmd := exec.CommandContext(ctx, "checkupdates")
	output, err := cmd.Output()
	if err != nil {
		// Exit code 2 in checkupdates means "no updates available"
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 2 {
			return []PackageUpdate{}, nil
		}
		return nil, err
	}

	return parseCheckupdatesOutput(output), nil
}

// parseCheckupdatesOutput parses the stdout lines from checkupdates into structured PackageUpdate slice.
func parseCheckupdatesOutput(output []byte) []PackageUpdate {
	var updates []PackageUpdate
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		// Format: <name> <old> -> <new>
		parts := strings.Fields(line)
		if len(parts) >= 4 && parts[2] == "->" {
			pkgName := parts[0]
			updates = append(updates, PackageUpdate{
				Name:       pkgName,
				OldVersion: parts[1],
				NewVersion: parts[3],
				IsCritical: CriticalPackages[pkgName],
			})
		}
	}
	return updates
}

// GetOrphans lists unneeded dependency packages with their metadata.
func GetOrphans(ctx context.Context) ([]OrphanPackage, error) {
	cmd := exec.CommandContext(ctx, "pacman", "-Qtdq")
	output, err := cmd.Output()
	if err != nil {
		// If exit code is 1, it usually means no orphan packages found
		return []OrphanPackage{}, nil
	}

	var orphanNames []string
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		name := strings.TrimSpace(scanner.Text())
		if name != "" {
			orphanNames = append(orphanNames, name)
		}
	}

	if len(orphanNames) == 0 {
		return []OrphanPackage{}, nil
	}

	// Security fix: pass "--" to prevent flag injection if an orphan name starts with "-"
	// Also batch into chunks of 100 to prevent exceeding ARG_MAX
	const batchSize = 100
	var allOrphans []OrphanPackage

	for i := 0; i < len(orphanNames); i += batchSize {
		end := i + batchSize
		if end > len(orphanNames) {
			end = len(orphanNames)
		}
		chunk := orphanNames[i:end]

		args := append([]string{"-Qi", "--"}, chunk...)
		infoCmd := exec.CommandContext(ctx, "pacman", args...)
		infoOut, err := infoCmd.Output()
		if err != nil {
			for _, name := range chunk {
				allOrphans = append(allOrphans, OrphanPackage{Name: name})
			}
			continue
		}
		allOrphans = append(allOrphans, parsePacmanInfo(infoOut)...)
	}

	return allOrphans, nil
}

func parsePacmanInfo(output []byte) []OrphanPackage {
	var list []OrphanPackage
	var current OrphanPackage

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if current.Name != "" {
				list = append(list, current)
				current = OrphanPackage{}
			}
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		switch key {
		case "Name":
			current.Name = val
		case "Description":
			current.Description = val
		case "Installed Size":
			current.Size = val
		}
	}

	if current.Name != "" {
		list = append(list, current)
	}

	return list
}

// GetCacheSize returns a human-readable size of /var/cache/pacman/pkg.
func GetCacheSize(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "du", "-sh", "/var/cache/pacman/pkg")
	output, err := cmd.Output()
	if err != nil {
		return "Unknown", err
	}
	parts := strings.Fields(string(output))
	if len(parts) > 0 {
		return parts[0], nil
	}
	return "0B", nil
}

// GetPacnewFiles searches for unmerged .pacnew configuration files under /etc up to 3 levels deep.
func GetPacnewFiles() ([]string, error) {
	var pacnewFiles []string
	const maxDepth = 3

	err := filepath.WalkDir("/etc", func(path string, d fs.DirEntry, err error) error {
		// Gracefully skip unreadable directories without failing the entire walk
		if err != nil {
			if os.IsPermission(err) {
				return fs.SkipDir
			}
			return nil
		}

		// Calculate current depth relative to /etc
		rel, err := filepath.Rel("/etc", path)
		if err != nil {
			return nil
		}
		depth := strings.Count(rel, string(filepath.Separator))
		if depth > maxDepth {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		if !d.IsDir() && strings.HasSuffix(d.Name(), ".pacnew") {
			pacnewFiles = append(pacnewFiles, path)
		}
		return nil
	})

	return pacnewFiles, err
}
