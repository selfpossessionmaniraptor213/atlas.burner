# Atlas Burner

![Banner Image](./banner-image.png)

**atlas.burner** is a beautiful, interactive TUI application for downloading OS images and burning them to USB drives. Part of the **Atlas Suite**.

![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)
![Platform](https://img.shields.io/badge/platform-Windows%20%7C%20Linux%20%7C%20macOS-lightgrey)

> [!WARNING]
> **This is an experimental tool.** It writes directly to raw block devices and can permanently destroy data. Use at your own discretion. Always double-check the target device before confirming a burn. The authors are not responsible for any data loss.

## ✨ Features

- 🖥️ **OS Catalog:** Browse and download popular Linux, Windows, and BSD distributions directly from the TUI.
- 💾 **USB Detection:** Automatically detect connected removable USB devices with boot type info (MBR/GPT).
- 🔥 **Direct Burn:** Write ISO/IMG files directly to your selected USB drive.
- 🔧 **UEFI Boot Support:** Create UEFI-bootable USB drives with GPT + FAT32 EFI System Partition.
- 💿 **Format Drive:** Format USB drives with MBR or GPT partition tables and FAT32/NTFS/exFAT filesystems.
- ⚙️ **Burn Options:** Configure block size, partition table type, volume label, and post-burn verification.
- ⌨️ **Vim Bindings:** Navigate using `j`/`k` or arrow keys.
- 📦 **Cross-Platform:** Binaries available for Windows, Linux, and macOS.

## 🚀 Installation

### From Source
```bash
git clone https://github.com/fezcode/atlas.burner
cd atlas.burner
gobake build
```

## ⌨️ Usage

Run the binary to enter the TUI. Requires administrator/root privileges to write to block devices:
```bash
sudo ./atlas.burner
# or run as Administrator on Windows (UAC prompt is automatic)
```

### Modes

| Mode | Description |
|---|---|
| **Browse OS Catalog** | Pick from a curated list of distributions, download, and burn. |
| **Burn Local Image** | Select an ISO or IMG file already on your machine to burn. |
| **Format Drive** | Format the USB drive with a fresh partition table and filesystem. |

### Partition Table Options

| Option | Description |
|---|---|
| **Keep original** | Write the image byte-for-byte (raw `dd` mode). Best for ISOs with their own boot structures. |
| **MBR** | Create an MBR partition table before burning. |
| **UEFI (GPT/FAT32)** | Extract ISO contents into a GPT disk with a FAT32 EFI System Partition for UEFI boot. |

### Command-line Flags
```
atlas.burner              Start the interactive TUI
atlas.burner -v           Show version
atlas.burner -h           Show help
```

## 📄 License
MIT License - see [LICENSE](LICENSE) for details.
