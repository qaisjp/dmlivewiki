# dmlivewiki

This has only been tested on Windows, but it *probably* works on other systems.

Use `dmlivewiki help` for help.

## Commands

- `dmlivewiki generate --tour "<tour name>" <directory>
    - Generates an information file (`.txt`) of each album in a given directory. The information file will contain the name of the tour that has been given.
- `dmlivewiki checksum <directory>`
    - Performs a checksum of each album in the given directory, placing a `.ffp` and `.md5` in each album.
- `dmlivewiki verify <directory>`
    - Verifies the contents of files listed in the `.md5` file.
- `dmlivewiki wiki <directory>`
    - Generates a `.wiki` file of each album in a given directory. The information in the wiki file is derived from the data in the corresponding "information file".
    - The filename is dervied from the "Album" field, which is also available in the "information file".

# Requires
This only requires `metaflac` to be on the system. You can obtain this from [xiph.org](https://xiph.org/flac/download.html), but the binary (distributed under the GPL) is distributed in the release zip.

- On Debian/Ubuntu/whatever you can use `apt install flac` to get `metaflac`.
- On macOS use `brew install flac`