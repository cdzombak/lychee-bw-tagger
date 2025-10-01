# lychee-bw-tagger

Automatically detects and tags grayscale photos in your Lychee installation.

## What It Does

Analyzes photos in a [Lychee](https://github.com/LycheeOrg/Lychee) database to identify grayscale images and automatically tags them as "Black & White". The program:

1. Adds a `_dz_bw` column to the photos table to track processing
2. Finds or creates a "Black & White" tag in the database
3. Downloads and analyzes unprocessed photos using grayscale detection
4. Applies the tag to detected grayscale photos

Supports JPEG, PNG, and WebP formats. Automatically skips videos and RAW files.

## Usage

### Configuration

Create `config.yml` based on `config.yml.example`:

```yaml
database:
  host: localhost
  port: 3306
  username: lychee_user
  password: your_password_here
  database: lychee

grayscale_tolerance: 0.1
image_base_url: "https://your-lychee-instance.com"
```

### Running

```bash
lychee-bw-tagger
lychee-bw-tagger -config /path/to/config.yml
lychee-bw-tagger -verbose
```

### Docker

```shell
docker run --rm -v /path/to/config.yml:/config.yml cdzombak/lychee-bw-tagger:1 -config /config.yml
```

## Installation

## Debian via apt repository

Set up my `oss` apt repository:

```shell
sudo mkdir -p /etc/apt/keyrings
curl -fsSL https://dist.cdzombak.net/keys/dist-cdzombak-net.gpg -o /etc/apt/keyrings/dist-cdzombak-net.gpg
sudo chmod 644 /etc/apt/keyrings/dist-cdzombak-net.gpg
sudo mkdir -p /etc/apt/sources.list.d
sudo curl -fsSL https://dist.cdzombak.net/cdzombak-oss.sources -o /etc/apt/sources.list.d/cdzombak-oss.sources
sudo chmod 644 /etc/apt/sources.list.d/cdzombak-oss.sources
sudo apt update
```

Then install `lychee-bw-tagger` via `apt-get`:

```shell
sudo apt-get install lychee-bw-tagger
```

## Homebrew

```shell
brew install cdzombak/oss/lychee-bw-tagger
```

## Manual from build artifacts

Pre-built binaries for Linux and macOS on various architectures are downloadable from each [GitHub Release](https://github.com/cdzombak/lychee-bw-tagger/releases). Debian packages for each release are available as well.

## License

GNU GPL v3; see [LICENSE](LICENSE) in this repo for details.

## Author

Chris Dzombak
- [dzombak.com](https://www.dzombak.com)
- [GitHub @cdzombak](https://github.com/cdzombak)
