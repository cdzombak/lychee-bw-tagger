# Lychee Black & White Tagger

Automatically detects and tags grayscale photos in a Lychee installation.

## Configuration

Copy `config.yml.example` to `config.yml`:

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

## Usage

```bash
./lychee-bw-tagger
./lychee-bw-tagger -config /path/to/config.yml
./lychee-bw-tagger -verbose
```

## Building

```bash
go build -o lychee-bw-tagger
```

## How It Works

1. Adds `_dz_bw` column to photos table
2. Finds or creates "Black & White" tag
3. Downloads and analyzes unprocessed photos
4. Tags grayscale photos and updates database

Supports JPEG, PNG, and WebP. Skips videos and RAW files.
