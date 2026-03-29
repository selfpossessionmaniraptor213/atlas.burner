//go:build windows

package usb

import (
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

const ioctlDiskUpdateProperties = 0x00070140

// finalizeFormat rescans the disk, assigns a drive letter, and formats if needed.
// Uses PowerShell storage cmdlets which work on removable media (diskpart doesn't).
func finalizeFormat(devicePath string, fsType string, label string) error {
	diskNum, err := parseDiskNumber(devicePath)
	if err != nil {
		return err
	}

	// Rescan the disk so Windows picks up the new partition table
	rescanDisk(devicePath)
	time.Sleep(2 * time.Second)

	// Use PowerShell to assign a drive letter
	letter, err := psAssignDriveLetter(diskNum)
	if err != nil {
		return fmt.Errorf("failed to assign drive letter: %v", err)
	}

	if fsType == "fat32" {
		// FAT32 was already written directly — done
		return nil
	}

	// Format with NTFS/exFAT using PowerShell Format-Volume
	return psFormatVolume(letter, fsType, label)
}

func rescanDisk(devicePath string) {
	pathPtr, _ := syscall.UTF16PtrFromString(devicePath)
	handle, _, _ := procCreateFileW.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		syscall.GENERIC_READ|syscall.GENERIC_WRITE,
		syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE,
		0,
		syscall.OPEN_EXISTING,
		0,
		0,
	)
	if handle == uintptr(syscall.InvalidHandle) {
		return
	}
	defer syscall.CloseHandle(syscall.Handle(handle))

	var bytesReturned uint32
	procDeviceIoControl.Call(
		handle,
		ioctlDiskUpdateProperties,
		0, 0,
		0, 0,
		uintptr(unsafe.Pointer(&bytesReturned)),
		0,
	)
}

// psAssignDriveLetter uses PowerShell to assign a drive letter to partition 1 on the disk.
// Returns the assigned drive letter.
func psAssignDriveLetter(diskNum uint32) (string, error) {
	// First try to get existing drive letter
	script := fmt.Sprintf(
		`$p = Get-Partition -DiskNumber %d -PartitionNumber 1 -ErrorAction SilentlyContinue; `+
			`if ($p -and $p.DriveLetter -and $p.DriveLetter -ne [char]0) { `+
			`  $p.DriveLetter `+
			`} else { `+
			`  $p | Add-PartitionAccessPath -AssignDriveLetter -ErrorAction Stop; `+
			`  Start-Sleep -Seconds 1; `+
			`  (Get-Partition -DiskNumber %d -PartitionNumber 1).DriveLetter `+
			`}`,
		diskNum, diskNum,
	)

	out, err := runPowerShell(script)
	if err != nil {
		return "", fmt.Errorf("assign drive letter: %v\n%s", err, out)
	}

	letter := strings.TrimSpace(out)
	if len(letter) == 1 && letter[0] >= 'A' && letter[0] <= 'Z' {
		return letter, nil
	}
	// Lowercase check
	if len(letter) == 1 && letter[0] >= 'a' && letter[0] <= 'z' {
		return strings.ToUpper(letter), nil
	}

	return "", fmt.Errorf("unexpected drive letter response: %q", letter)
}

// psFormatVolume formats a volume using PowerShell Format-Volume.
func psFormatVolume(driveLetter string, fsType string, label string) error {
	fs := strings.ToUpper(fsType)
	// exfat -> exFAT for PowerShell
	if fs == "EXFAT" {
		fs = "exFAT"
	}

	script := fmt.Sprintf(
		`Format-Volume -DriveLetter %s -FileSystem %s -NewFileSystemLabel "%s" -Confirm:$false -Force`,
		driveLetter, fs, label,
	)

	out, err := runPowerShell(script)
	if err != nil {
		return fmt.Errorf("format failed: %v\n%s", err, out)
	}
	return nil
}

func runPowerShell(script string) (string, error) {
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	output, err := cmd.CombinedOutput()
	return string(output), err
}
