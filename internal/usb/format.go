package usb

import (
	"encoding/binary"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
)

// FormatDevice formats a USB drive with the specified partition table and filesystem.
// FAT32 is written directly; NTFS and exFAT use platform-specific formatting tools.
func FormatDevice(devicePath string, deviceSize uint64, partType string, fsType string, label string, progressCallback func(float64)) error {
	report := func(p float64) {
		if progressCallback != nil {
			progressCallback(p)
		}
	}

	if deviceSize == 0 {
		return fmt.Errorf("cannot format: device size is unknown")
	}

	if label == "" {
		label = "ATLAS"
	}

	report(0.05)

	dev, err := openDeviceForWrite(devicePath)
	if err != nil {
		return err
	}

	report(0.10)

	totalSectors := deviceSize / 512
	partStartLBA := uint64(2048)
	partEndLBA := totalSectors - 1

	switch partType {
	case "gpt":
		partEndLBA = totalSectors - 34
		err = writeGPT(dev, totalSectors, partStartLBA, partEndLBA)
	case "mbr":
		err = writeMBR(dev, partStartLBA, partEndLBA)
	default:
		return fmt.Errorf("unsupported partition type: %s", partType)
	}
	if err != nil {
		dev.Close()
		return fmt.Errorf("failed to write partition table: %v", err)
	}

	report(0.30)

	if fsType == "fat32" {
		// FAT32: write structures directly
		volLabel := formatVolumeLabel(label)
		partSizeSectors := partEndLBA - partStartLBA + 1
		err = writeFAT32(dev, partStartLBA, partSizeSectors, volLabel)
		dev.Close()
		if err != nil {
			return fmt.Errorf("failed to create FAT32: %v", err)
		}
	} else {
		// NTFS/exFAT: close device first, then use platform tools
		dev.Close()
	}

	report(0.50)

	// Rescan disk so OS recognizes new partition table, then format if needed
	err = finalizeFormat(devicePath, fsType, label)
	if err != nil {
		return err
	}

	report(1.0)
	return nil
}

func formatVolumeLabel(label string) [11]byte {
	var vol [11]byte
	upper := strings.ToUpper(label)
	for i := 0; i < 11; i++ {
		if i < len(upper) {
			vol[i] = upper[i]
		} else {
			vol[i] = ' '
		}
	}
	return vol
}

// writeGPT writes a protective MBR + primary GPT header + partition entries + backup GPT.
func writeGPT(dev *os.File, totalSectors, partStartLBA, partEndLBA uint64) error {
	// 1. Write protective MBR at LBA 0
	pmbr := make([]byte, 512)
	// Partition entry 1 at offset 446: protective MBR entry
	pe := pmbr[446:]
	pe[0] = 0x00                                                  // not bootable
	pe[1], pe[2], pe[3] = 0x00, 0x02, 0x00                       // CHS start
	pe[4] = 0xEE                                                  // GPT protective
	pe[5], pe[6], pe[7] = 0xFF, 0xFF, 0xFF                       // CHS end
	binary.LittleEndian.PutUint32(pe[8:12], 1)                    // LBA start
	size := uint32(totalSectors - 1)
	if totalSectors-1 > 0xFFFFFFFF {
		size = 0xFFFFFFFF
	}
	binary.LittleEndian.PutUint32(pe[12:16], size)
	pmbr[510] = 0x55
	pmbr[511] = 0xAA
	if _, err := dev.WriteAt(pmbr, 0); err != nil {
		return fmt.Errorf("write protective MBR: %v", err)
	}

	// 2. Build partition entry array (128 entries * 128 bytes = 16384 bytes)
	const partEntrySize = 128
	const partArrayCount = 128
	partArray := make([]byte, partEntrySize*partArrayCount)

	// Single EFI System Partition entry
	entry := partArray[0:partEntrySize]
	// Partition type GUID: C12A7328-F81F-11D2-BA4B-00A0C93EC93B (EFI System)
	typeGUID := mixedEndianGUID([16]byte{0xC1, 0x2A, 0x73, 0x28, 0xF8, 0x1F, 0x11, 0xD2, 0xBA, 0x4B, 0x00, 0xA0, 0xC9, 0x3E, 0xC9, 0x3B})
	copy(entry[0:16], typeGUID[:])
	// Unique partition GUID
	uniqueGUID := newRandomGUID()
	copy(entry[16:32], uniqueGUID[:])
	binary.LittleEndian.PutUint64(entry[32:40], partStartLBA)
	binary.LittleEndian.PutUint64(entry[40:48], partEndLBA)
	// Attributes: 0
	// Name: "EFI System" in UTF-16LE
	name := utf16LEName("EFI System")
	copy(entry[56:128], name)

	partArrayCRC := crc32Bytes(partArray)

	// 3. Write primary GPT header at LBA 1
	lastUsableLBA := totalSectors - 34
	primaryHeader := buildGPTHeader(1, totalSectors-1, lastUsableLBA, 2, partArrayCRC)
	if _, err := dev.WriteAt(primaryHeader, 512); err != nil {
		return fmt.Errorf("write primary GPT header: %v", err)
	}

	// 4. Write primary partition array at LBA 2
	if _, err := dev.WriteAt(partArray, 2*512); err != nil {
		return fmt.Errorf("write primary partition array: %v", err)
	}

	// 5. Write backup partition array at LBA (totalSectors - 33)
	backupPartLBA := totalSectors - 33
	if _, err := dev.WriteAt(partArray, int64(backupPartLBA)*512); err != nil {
		return fmt.Errorf("write backup partition array: %v", err)
	}

	// 6. Write backup GPT header at last LBA
	backupHeader := buildGPTHeader(totalSectors-1, 1, lastUsableLBA, backupPartLBA, partArrayCRC)
	if _, err := dev.WriteAt(backupHeader, int64(totalSectors-1)*512); err != nil {
		return fmt.Errorf("write backup GPT header: %v", err)
	}

	return nil
}

func buildGPTHeader(myLBA, altLBA, lastUsableLBA, partArrayLBA uint64, partArrayCRC uint32) []byte {
	hdr := make([]byte, 512)

	// Signature: "EFI PART"
	copy(hdr[0:8], []byte("EFI PART"))
	// Revision: 1.0
	binary.LittleEndian.PutUint32(hdr[8:12], 0x00010000)
	// Header size: 92
	binary.LittleEndian.PutUint32(hdr[12:16], 92)
	// Header CRC32: computed after filling everything else
	// Reserved: 0
	binary.LittleEndian.PutUint64(hdr[24:32], myLBA)
	binary.LittleEndian.PutUint64(hdr[32:40], altLBA)
	binary.LittleEndian.PutUint64(hdr[40:48], 2048)             // first usable LBA
	binary.LittleEndian.PutUint64(hdr[48:56], lastUsableLBA)    // last usable LBA
	diskGUID := newRandomGUID()
	copy(hdr[56:72], diskGUID[:])
	binary.LittleEndian.PutUint64(hdr[72:80], partArrayLBA)     // partition entries start LBA
	binary.LittleEndian.PutUint32(hdr[80:84], 128)              // number of partition entries
	binary.LittleEndian.PutUint32(hdr[84:88], 128)              // size of each partition entry
	binary.LittleEndian.PutUint32(hdr[88:92], partArrayCRC)     // CRC32 of partition array

	// Compute header CRC32 (over first 92 bytes, with CRC field zeroed)
	binary.LittleEndian.PutUint32(hdr[16:20], 0)
	headerCRC := crc32Bytes(hdr[0:92])
	binary.LittleEndian.PutUint32(hdr[16:20], headerCRC)

	return hdr
}

// writeMBR writes an MBR partition table with one FAT32 LBA partition.
func writeMBR(dev *os.File, partStartLBA, partEndLBA uint64) error {
	mbr := make([]byte, 512)

	pe := mbr[446:]
	pe[0] = 0x80                                              // bootable
	pe[1], pe[2], pe[3] = 0xFE, 0xFF, 0xFF                   // CHS start (LBA mode)
	pe[4] = 0x0C                                              // FAT32 LBA
	pe[5], pe[6], pe[7] = 0xFE, 0xFF, 0xFF                   // CHS end (LBA mode)
	binary.LittleEndian.PutUint32(pe[8:12], uint32(partStartLBA))
	binary.LittleEndian.PutUint32(pe[12:16], uint32(partEndLBA-partStartLBA+1))

	mbr[510] = 0x55
	mbr[511] = 0xAA

	_, err := dev.WriteAt(mbr, 0)
	return err
}

// writeFAT32 writes a FAT32 BPB, FSInfo, backup boot sector, and two FAT copies.
func writeFAT32(dev *os.File, partStartLBA, partSizeSectors uint64, volLabel [11]byte) error {
	// Calculate FAT32 parameters
	bytesPerSector := uint16(512)
	sectorsPerCluster := chooseSectorsPerCluster(partSizeSectors * 512)
	reservedSectors := uint16(32)
	numFATs := uint8(2)

	// Calculate FAT size
	// Total data sectors = partSizeSectors - reservedSectors - (numFATs * fatSizeSectors)
	// Total clusters = dataSectors / sectorsPerCluster
	// FAT entries = totalClusters + 2 (for reserved entries)
	// FAT size in sectors = ceil(fatEntries * 4 / 512)
	// Solve iteratively:
	dataSectors := partSizeSectors - uint64(reservedSectors)
	totalClusters := dataSectors / uint64(sectorsPerCluster)
	fatEntries := totalClusters + 2
	fatSizeSectors := (fatEntries*4 + 511) / 512
	// Adjust: subtract FAT space from data
	dataSectors = partSizeSectors - uint64(reservedSectors) - 2*fatSizeSectors
	totalClusters = dataSectors / uint64(sectorsPerCluster)

	// Build boot sector (BPB)
	bs := make([]byte, 512)
	bs[0] = 0xEB // JMP short
	bs[1] = 0x58 // offset
	bs[2] = 0x90 // NOP
	copy(bs[3:11], []byte("ATLAS   "))                            // OEM name
	binary.LittleEndian.PutUint16(bs[11:13], bytesPerSector)      // bytes per sector
	bs[13] = sectorsPerCluster                                     // sectors per cluster
	binary.LittleEndian.PutUint16(bs[14:16], reservedSectors)     // reserved sectors
	bs[16] = numFATs                                               // number of FATs
	binary.LittleEndian.PutUint16(bs[17:19], 0)                   // root entry count (0 for FAT32)
	binary.LittleEndian.PutUint16(bs[19:21], 0)                   // total sectors 16 (0 for FAT32)
	bs[21] = 0xF8                                                  // media type (fixed disk)
	binary.LittleEndian.PutUint16(bs[22:24], 0)                   // FAT size 16 (0 for FAT32)
	binary.LittleEndian.PutUint16(bs[24:26], 63)                  // sectors per track
	binary.LittleEndian.PutUint16(bs[26:28], 255)                 // number of heads
	binary.LittleEndian.PutUint32(bs[28:32], uint32(partStartLBA)) // hidden sectors
	binary.LittleEndian.PutUint32(bs[32:36], uint32(partSizeSectors)) // total sectors 32

	// FAT32-specific fields (offset 36+)
	binary.LittleEndian.PutUint32(bs[36:40], uint32(fatSizeSectors)) // FAT size 32
	binary.LittleEndian.PutUint16(bs[40:42], 0)                     // ext flags
	binary.LittleEndian.PutUint16(bs[42:44], 0)                     // FS version
	binary.LittleEndian.PutUint32(bs[44:48], 2)                     // root cluster
	binary.LittleEndian.PutUint16(bs[48:50], 1)                     // FSInfo sector
	binary.LittleEndian.PutUint16(bs[50:52], 6)                     // backup boot sector
	// bytes 52-63: reserved (zeros)
	bs[64] = 0x80                                                    // drive number
	bs[66] = 0x29                                                    // boot sig
	// Volume serial number (random)
	serial := newRandomSerial()
	binary.LittleEndian.PutUint32(bs[67:71], serial)
	copy(bs[71:82], volLabel[:])                                     // volume label
	copy(bs[82:90], []byte("FAT32   "))                              // FS type
	bs[510] = 0x55
	bs[511] = 0xAA

	// Write boot sector
	partOffset := int64(partStartLBA) * 512
	if _, err := dev.WriteAt(bs, partOffset); err != nil {
		return fmt.Errorf("write boot sector: %v", err)
	}

	// Write FSInfo sector (sector 1 of partition)
	fsinfo := make([]byte, 512)
	binary.LittleEndian.PutUint32(fsinfo[0:4], 0x41615252)       // FSInfo signature 1
	binary.LittleEndian.PutUint32(fsinfo[484:488], 0x61417272)   // FSInfo signature 2
	binary.LittleEndian.PutUint32(fsinfo[488:492], uint32(totalClusters-1)) // free clusters
	binary.LittleEndian.PutUint32(fsinfo[492:496], 3)            // next free cluster
	binary.LittleEndian.PutUint32(fsinfo[508:512], 0xAA550000)   // trail signature
	fsinfo[510] = 0x55
	fsinfo[511] = 0xAA
	if _, err := dev.WriteAt(fsinfo, partOffset+512); err != nil {
		return fmt.Errorf("write FSInfo: %v", err)
	}

	// Write backup boot sector at sector 6
	if _, err := dev.WriteAt(bs, partOffset+6*512); err != nil {
		return fmt.Errorf("write backup boot sector: %v", err)
	}
	// Write backup FSInfo at sector 7
	if _, err := dev.WriteAt(fsinfo, partOffset+7*512); err != nil {
		return fmt.Errorf("write backup FSInfo: %v", err)
	}

	// Write FAT tables
	// First 2 entries of FAT: entry 0 = media byte, entry 1 = EOC, entry 2 = EOC (root dir)
	fatStart := make([]byte, 512)
	binary.LittleEndian.PutUint32(fatStart[0:4], 0x0FFFFFF8)   // FAT entry 0 (media type)
	binary.LittleEndian.PutUint32(fatStart[4:8], 0x0FFFFFFF)   // FAT entry 1 (EOC)
	binary.LittleEndian.PutUint32(fatStart[8:12], 0x0FFFFFFF)  // FAT entry 2 (root dir, EOC)

	// FAT 1
	fat1Offset := partOffset + int64(reservedSectors)*512
	if _, err := dev.WriteAt(fatStart, fat1Offset); err != nil {
		return fmt.Errorf("write FAT1: %v", err)
	}

	// FAT 2
	fat2Offset := fat1Offset + int64(fatSizeSectors)*512
	if _, err := dev.WriteAt(fatStart, fat2Offset); err != nil {
		return fmt.Errorf("write FAT2: %v", err)
	}

	// Zero out root directory cluster (first cluster after FATs)
	rootDirOffset := fat2Offset + int64(fatSizeSectors)*512
	rootDir := make([]byte, uint64(sectorsPerCluster)*512)

	// Write volume label entry in root directory
	copy(rootDir[0:11], volLabel[:])
	rootDir[11] = 0x08 // attribute: volume label

	if _, err := dev.WriteAt(rootDir, rootDirOffset); err != nil {
		return fmt.Errorf("write root directory: %v", err)
	}

	return nil
}

// chooseSectorsPerCluster picks sectors per cluster based on volume size.
// Follows Microsoft's recommended defaults.
func chooseSectorsPerCluster(volumeBytes uint64) uint8 {
	gb := volumeBytes / (1024 * 1024 * 1024)
	switch {
	case gb < 1:
		return 8 // 4 KB clusters
	case gb < 8:
		return 8 // 4 KB clusters
	case gb < 16:
		return 16 // 8 KB clusters
	case gb < 32:
		return 32 // 16 KB clusters
	default:
		return 64 // 32 KB clusters
	}
}

func newRandomSerial() uint32 {
	u := uuid.New()
	return binary.LittleEndian.Uint32(u[:4])
}

// mixedEndianGUID converts a standard GUID byte array to mixed-endian format
// as used in GPT (first 3 groups are little-endian, last 2 are big-endian).
func mixedEndianGUID(guid [16]byte) [16]byte {
	var out [16]byte
	// Group 1 (4 bytes) - little endian
	out[0] = guid[3]
	out[1] = guid[2]
	out[2] = guid[1]
	out[3] = guid[0]
	// Group 2 (2 bytes) - little endian
	out[4] = guid[5]
	out[5] = guid[4]
	// Group 3 (2 bytes) - little endian
	out[6] = guid[7]
	out[7] = guid[6]
	// Groups 4 & 5 (8 bytes) - big endian (no swap)
	copy(out[8:16], guid[8:16])
	return out
}

func newRandomGUID() [16]byte {
	u := uuid.New()
	return mixedEndianGUID(u)
}

func utf16LEName(name string) []byte {
	buf := make([]byte, 72) // max 36 UTF-16 chars = 72 bytes
	for i, r := range name {
		if i >= 36 {
			break
		}
		binary.LittleEndian.PutUint16(buf[i*2:(i+1)*2], uint16(r))
	}
	return buf
}

func crc32Bytes(data []byte) uint32 {
	// IEEE CRC-32
	var table [256]uint32
	for i := 0; i < 256; i++ {
		crc := uint32(i)
		for j := 0; j < 8; j++ {
			if crc&1 != 0 {
				crc = (crc >> 1) ^ 0xEDB88320
			} else {
				crc >>= 1
			}
		}
		table[i] = crc
	}

	crc := uint32(0xFFFFFFFF)
	for _, b := range data {
		crc = table[byte(crc)^b] ^ (crc >> 8)
	}
	return crc ^ 0xFFFFFFFF
}
