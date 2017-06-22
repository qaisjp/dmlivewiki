# dmlivewiki

This has only been tested on Windows, but it *probably* works on other systems.

Use `dmlivewiki help` for help.

## Commands

- `dmlivewiki checksum <directory>`
    - Performs a checksum of each album in the given directory, placing a `.ffp` and `.md5` in each album.

# Requires
This only requires `metaflac` to be on the system. You can obtain this from [xiph.org](https://xiph.org/flac/download.html), but the binary (distributed under the GPL) is distributed in the release zip.

- On Debian/Ubuntu/whatever you can use `apt install flac` to get `metaflac`.
- On macOS use `brew install flac`