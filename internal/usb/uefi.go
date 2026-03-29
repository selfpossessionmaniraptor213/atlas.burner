package usb

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	pathpkg "path"
	"strings"

	diskfs "github.com/diskfs/go-diskfs"
	diskpkg "github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/diskfs/go-diskfs/partition/gpt"
)

const (
	uefiOverheadBytes  = 256 * 1024 * 1024 // 256 MB overhead for GPT + FAT32 structures
	sectorSize         = 512
	partitionStartLBA  = 2048 // standard GPT first partition start
	fat32MaxFileSize   = 4*1024*1024*1024 - 1
)

// BurnUEFI creates a UEFI-bootable USB by extracting ISO contents into a
// GPT disk image with a FAT32 EFI System Partition, then writing it to the device.
func BurnUEFI(isoPath, devicePath string, deviceSize uint64, progressCallback func(float64)) error {
	report := func(p float64) {
		if progressCallback != nil {
			progressCallback(p)
		}
	}

	// Phase 1: Read ISO filesystem and collect file manifest (0-5%)
	report(0)
	isoDisk, err := diskfs.Open(isoPath, diskfs.WithOpenMode(diskfs.ReadOnly))
	if err != nil {
		return fmt.Errorf("failed to open ISO: %v", err)
	}

	isoFS, err := isoDisk.GetFilesystem(0)
	if err != nil {
		return fmt.Errorf("failed to read ISO filesystem: %v", err)
	}

	var manifest []fileEntry
	var totalSize int64
	err = walkISO(isoFS, ".", &manifest, &totalSize)
	if err != nil {
		return fmt.Errorf("failed to scan ISO contents: %v", err)
	}

	// Check FAT32 file size limit
	for _, entry := range manifest {
		if !entry.isDir && entry.size > fat32MaxFileSize {
			return fmt.Errorf("file %s is %.1f GB, exceeds FAT32 4GB limit — use raw dd mode (Keep original) instead",
				entry.path, float64(entry.size)/(1024*1024*1024))
		}
	}
	report(0.05)

	// Phase 2: Create temp disk image with GPT + FAT32 (5-10%)
	imageSize := totalSize + uefiOverheadBytes
	if int64(deviceSize) > 0 && imageSize > int64(deviceSize) {
		return fmt.Errorf("ISO contents (%.1f GB) exceed USB capacity (%.1f GB)",
			float64(totalSize)/(1024*1024*1024), float64(deviceSize)/(1024*1024*1024))
	}

	// Reserve a temp path, then remove so diskfs.Create can create it fresh
	tmpFile, err := os.CreateTemp("", "atlas-uefi-*.img")
	if err != nil {
		return fmt.Errorf("failed to create temp image: %v", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	os.Remove(tmpPath)
	defer os.Remove(tmpPath)

	imgDisk, err := diskfs.Create(tmpPath, imageSize, diskfs.SectorSizeDefault)
	if err != nil {
		return fmt.Errorf("failed to create disk image: %v", err)
	}

	// Create GPT with single EFI System Partition
	partitionSectors := (imageSize - partitionStartLBA*sectorSize) / sectorSize
	// Leave room for backup GPT (34 sectors at end)
	partitionEnd := uint64(partitionStartLBA) + uint64(partitionSectors) - 34 - 1

	table := &gpt.Table{
		Partitions: []*gpt.Partition{
			{
				Index: 1,
				Start: uint64(partitionStartLBA),
				End:   partitionEnd,
				Type:  gpt.EFISystemPartition,
				Name:  "EFI System",
			},
		},
	}
	if err := imgDisk.Partition(table); err != nil {
		return fmt.Errorf("failed to create GPT partition table: %v", err)
	}

	fatFS, err := imgDisk.CreateFilesystem(diskpkg.FilesystemSpec{
		Partition:   1,
		FSType:      filesystem.TypeFat32,
		VolumeLabel: "EFI",
	})
	if err != nil {
		return fmt.Errorf("failed to create FAT32 filesystem: %v", err)
	}
	report(0.10)

	// Phase 3: Copy files from ISO to FAT32 (10-90%)
	var copiedBytes int64
	for _, entry := range manifest {
		if entry.isDir {
			if err := fatFS.Mkdir(entry.path); err != nil {
				// Ignore mkdir errors for existing dirs
				continue
			}
			continue
		}

		if err := copyFileToFAT32(isoFS, fatFS, entry.isoPath, entry.path); err != nil {
			return fmt.Errorf("failed to copy %s: %v", entry.path, err)
		}

		copiedBytes += entry.size
		progress := 0.10 + 0.80*(float64(copiedBytes)/float64(totalSize))
		report(progress)
	}
	report(0.90)

	// Phase 4: Burn temp image to device (90-100%)
	// Close the filesystem and disk to flush all writes
	if closer, ok := fatFS.(io.Closer); ok {
		closer.Close()
	}

	err = BurnImage(tmpPath, devicePath, func(p float64) {
		report(0.90 + 0.10*p)
	})
	if err != nil {
		return fmt.Errorf("failed to write image to device: %v", err)
	}

	return nil
}

type fileEntry struct {
	path    string // FAT32 destination path (with leading /)
	isoPath string // ISO source path (without leading /, go-diskfs format)
	size    int64
	isDir   bool
}

func walkISO(isoFS filesystem.FileSystem, dir string, manifest *[]fileEntry, totalSize *int64) error {
	entries, err := isoFS.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		name := entry.Name()
		// Skip . and .. entries
		if name == "." || name == ".." {
			continue
		}

		// Use path.Join (Unix-style) not filepath.Join (OS-specific) for ISO paths.
		// go-diskfs uses io/fs.ValidPath which requires paths without leading /.
		isoPath := pathpkg.Join(dir, name)

		// For the FAT32 destination, we need paths with leading /
		fatPath := "/" + isoPath
		if strings.HasPrefix(isoPath, "./") {
			fatPath = "/" + isoPath[2:]
		}
		if isoPath == "." {
			fatPath = "/"
		}

		if entry.IsDir() {
			*manifest = append(*manifest, fileEntry{path: fatPath, isoPath: isoPath, isDir: true})
			if err := walkISO(isoFS, isoPath, manifest, totalSize); err != nil {
				return err
			}
		} else {
			info, err := entry.Info()
			if err != nil {
				// Skip files we can't stat
				continue
			}
			size := info.Size()
			*manifest = append(*manifest, fileEntry{path: fatPath, isoPath: isoPath, size: size})
			*totalSize += size
		}
	}
	return nil
}

func copyFileToFAT32(srcFS, dstFS filesystem.FileSystem, isoPath string, fatPath string) error {
	// Ensure parent directory exists on the FAT32 side
	parentDir := pathpkg.Dir(fatPath)
	if parentDir != "/" && parentDir != "." {
		dstFS.Mkdir(parentDir) // ignore error, may already exist
	}

	// Use isoPath (no leading /) for reading from the ISO filesystem
	srcFile, err := srcFS.OpenFile(isoPath, os.O_RDONLY)
	if err != nil {
		// Try with fs.Open as fallback
		if opener, ok := srcFS.(interface{ Open(string) (fs.File, error) }); ok {
			f, err2 := opener.Open(isoPath)
			if err2 != nil {
				return fmt.Errorf("open source: %v (also tried Open: %v)", err, err2)
			}
			defer f.Close()
			return copyFromReader(dstFS, fatPath, f)
		}
		return fmt.Errorf("open source: %v", err)
	}
	defer srcFile.Close()

	return copyFromReader(dstFS, fatPath, srcFile)
}

func copyFromReader(dstFS filesystem.FileSystem, path string, src io.Reader) error {
	dstFile, err := dstFS.OpenFile(path, os.O_CREATE|os.O_RDWR)
	if err != nil {
		return fmt.Errorf("create destination: %v", err)
	}
	defer dstFile.Close()

	buf := make([]byte, 1024*1024) // 1MB copy buffer
	for {
		n, readErr := src.Read(buf)
		if n > 0 {
			if _, writeErr := dstFile.Write(buf[:n]); writeErr != nil {
				return fmt.Errorf("write: %v", writeErr)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("read: %v", readErr)
		}
	}

	return nil
}
