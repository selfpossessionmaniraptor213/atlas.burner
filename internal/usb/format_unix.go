//go:build !windows

package usb

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// finalizeFormat rescans the disk so Linux sees the new partition table,
// then formats with the chosen filesystem if needed, and attempts to mount.
func finalizeFormat(devicePath string, fsType string, label string) error {
	// Rescan partition table — try multiple tools since availability varies
	if path, err := exec.LookPath("partprobe"); err == nil {
		exec.Command(path, devicePath).Run()
	}
	if path, err := exec.LookPath("blockdev"); err == nil {
		exec.Command(path, "--rereadpt", devicePath).Run()
	}
	// Fallback: hdparm (works on some systems where others don't)
	if path, err := exec.LookPath("hdparm"); err == nil {
		exec.Command(path, "-z", devicePath).Run()
	}

	time.Sleep(1 * time.Second)

	if fsType == "fat32" {
		// FAT32 structures were already written directly — done
		return nil
	}

	// Find the first partition path
	partPath := resolvePartitionPath(devicePath)

	switch fsType {
	case "ntfs":
		mkfs, err := exec.LookPath("mkfs.ntfs")
		if err != nil {
			return fmt.Errorf("mkfs.ntfs not found — install ntfs-3g (e.g., sudo apt install ntfs-3g)")
		}
		cmd := exec.Command(mkfs, "-f", "-L", label, partPath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("mkfs.ntfs failed: %v\n%s", err, string(output))
		}

	case "exfat":
		// Try mkfs.exfat first, fall back to mkexfatfs
		mkfs, err := exec.LookPath("mkfs.exfat")
		if err != nil {
			mkfs, err = exec.LookPath("mkexfatfs")
		}
		if err != nil {
			return fmt.Errorf("mkfs.exfat not found — install exfatprogs (e.g., sudo apt install exfatprogs)")
		}
		cmd := exec.Command(mkfs, "-n", label, partPath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("mkfs.exfat failed: %v\n%s", err, string(output))
		}

	default:
		return fmt.Errorf("unsupported filesystem: %s", fsType)
	}

	return nil
}

// resolvePartitionPath returns the path to the first partition on a device.
// e.g., /dev/sdb -> /dev/sdb1, /dev/nvme0n1 -> /dev/nvme0n1p1
func resolvePartitionPath(devicePath string) string {
	if strings.Contains(devicePath, "nvme") || strings.Contains(devicePath, "loop") || strings.Contains(devicePath, "mmcblk") {
		return devicePath + "p1"
	}
	return devicePath + "1"
}
